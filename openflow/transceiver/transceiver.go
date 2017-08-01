/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service, Inc. All rights reserved.
 * Kitae Kim <superkkt@sds.co.kr>
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along
 * with this program; if not, write to the Free Software Foundation, Inc.,
 * 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
 */

package transceiver

import (
	"context"
	"encoding"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/superkkt/cherry/openflow"
	"github.com/superkkt/cherry/openflow/of10"
	"github.com/superkkt/cherry/openflow/of13"

	"github.com/pkg/errors"
	"github.com/superkkt/go-logging"
)

var (
	logger = logging.MustGetLogger("transceiver")
)

const (
	// Allowed idle time before we send an echo request to a switch.
	maxIdleTime = 10 * time.Second
	// I/O timeouts (These timeouts should be less than maxIdleTime).
	readTimeout  = 1 * time.Second
	writeTimeout = readTimeout * 2
)

type Writer interface {
	Write(msg encoding.BinaryMarshaler) error
}

type WriteCloser interface {
	Writer
	Close() error
}

type Transceiver struct {
	stream      *Stream
	observer    Handler
	version     uint8
	factory     openflow.Factory
	pingCounter uint
	closed      bool
}

type Handler interface {
	OnHello(openflow.Factory, Writer, openflow.Hello) error
	OnError(openflow.Factory, Writer, openflow.Error) error
	OnFeaturesReply(openflow.Factory, Writer, openflow.FeaturesReply) error
	OnGetConfigReply(openflow.Factory, Writer, openflow.GetConfigReply) error
	OnDescReply(openflow.Factory, Writer, openflow.DescReply) error
	OnPortDescReply(openflow.Factory, Writer, openflow.PortDescReply) error
	OnPortStatus(openflow.Factory, Writer, openflow.PortStatus) error
	OnFlowRemoved(openflow.Factory, Writer, openflow.FlowRemoved) error
	OnPacketIn(openflow.Factory, Writer, openflow.PacketIn) error
}

func NewTransceiver(stream *Stream, handler Handler) *Transceiver {
	if stream == nil {
		panic("stream is nil")
	}
	if handler == nil {
		panic("handler is nil")
	}

	return &Transceiver{
		stream:   stream,
		observer: handler,
	}
}

func (r *Transceiver) Version() (negotiated bool, version uint8) {
	if r.version == 0 {
		// Not yet negotiated
		return false, 0
	}

	return true, r.version
}

func isTimeout(err error) bool {
	type Timeout interface {
		Timeout() bool
	}

	if v, ok := err.(Timeout); ok {
		return v.Timeout()
	}

	return false
}

func (r *Transceiver) sendEchoRequest() error {
	if r.pingCounter > 2 {
		return errors.New("device does not respond to our echo request")
	}

	echo, err := r.factory.NewEchoRequest()
	if err != nil {
		return err
	}
	// We use current timestamp to check network latency between our controller and a switch.
	timestamp, err := time.Now().GobEncode()
	if err != nil {
		return err
	}
	echo.SetData(timestamp)

	if err := r.Write(echo); err != nil {
		return errors.Wrap(err, "failed to send ECHO_REQUEST message")
	}
	r.pingCounter++

	return nil
}

func (r *Transceiver) Run(ctx context.Context) error {
	defer logger.Info("transceiver is closed")
	r.stream.SetReadTimeout(readTimeout)
	r.stream.SetWriteTimeout(writeTimeout)

	readerCtx, cancelReader := context.WithCancel(ctx)
	defer cancelReader()
	reader := r.runReader(readerCtx)

	// Negotiate the protocol version
	packet, err := r.negotiate(ctx, reader)
	if err != nil {
		return errors.Wrap(err, "failed to negotiate the protocol version")
	}

	// Infinite loop
	for {
		// Dispatch the incoming packet
		if err := r.dispatch(packet); err != nil {
			if !isTemporaryErr(err) {
				return err
			}
			// Ignore the temporary error. Just log the error and keep go on.
			logger.Errorf("failed to dispatch the packet: %v", err)
		}

		// Read the next packet
		var ok bool
		select {
		case <-ctx.Done():
			logger.Info("context done")
			return nil
		case packet, ok = <-reader:
			if !ok {
				logger.Info("the reader channel is closed")
				return nil
			}
			remain := len(reader)
			if remain > 0 {
				logger.Debugf("%v remaining unread packet(s) in the reader channel", remain)
			}
		}
	}
}

func (r *Transceiver) negotiate(ctx context.Context, reader <-chan []byte) (packet []byte, err error) {
	select {
	case <-ctx.Done():
		return nil, errors.New("context done")
	case <-time.After(30 * time.Second):
		return nil, errors.New("inactive for too long")
	case packet, ok := <-reader:
		if !ok {
			return nil, errors.New("the reader channel is closed")
		}
		// The first message should be HELLO.
		if packet[1] != 0x00 {
			return nil, errors.New("missing HELLO message")
		}

		// Version negotiation
		if packet[0] < openflow.OF13_VERSION {
			r.version = openflow.OF10_VERSION
			r.factory = of10.NewFactory()
			logger.Info("negotiated to openflow version 1.0")
		} else {
			r.version = openflow.OF13_VERSION
			r.factory = of13.NewFactory()
			logger.Info("negotiated to openflow version 1.3")
		}

		// Return the initial packet to dispatch it.
		return packet, nil
	}
}

func (r *Transceiver) runReader(ctx context.Context) <-chan []byte {
	// Buffered channel
	c := make(chan []byte, 4096)
	go func() {
		// The channel c will be closed when this goroutine returns in order to notice the connection has been closed.
		defer close(c)
		defer logger.Info("transceiver reader is closed")

		lastActivated := time.Now()
		for {
			select {
			case <-ctx.Done():
				logger.Info("context done")
				return
			default:
			}

			// Read the next packet
			packet, err := r.readPacket()
			if err != nil {
				if !isTimeout(err) {
					logger.Errorf("failed to read the next packet: %v", err)
					return
				}
				// Timeout occurrs. Send a ping request if necessary.
				if time.Now().After(lastActivated.Add(maxIdleTime)) {
					if err := r.sendEchoRequest(); err != nil {
						logger.Errorf("failed to send an echo request: %v", err)
						return
					}
				}
				continue
			}
			// Update the timestamp
			lastActivated = time.Now()

			ok, err := r.handleEcho(packet)
			if err != nil {
				logger.Errorf("failed to handle the echo request or response: %v", err)
				return
			}
			if ok {
				// Do not forward the echo request and response
				// packets because this reader handles them.
				continue
			}

			// Forward messages except the echo request and response.
			select {
			case c <- packet:
			default:
				// Drop the packet if we cannot immediately carry it.
				logger.Error("transceiver buffer full: drop the incoming packet!")
			}
		}
	}()

	return c
}

func isTemporaryErr(err error) bool {
	e, ok := errors.Cause(err).(interface {
		Temporary() bool
	})
	return ok && e.Temporary()
}

func (r *Transceiver) readPacket() ([]byte, error) {
	header, err := r.stream.Peek(8) // peek ofp_header
	if err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint16(header[2:4])
	if length < 8 {
		return nil, openflow.ErrInvalidPacketLength
	}
	packet, err := r.stream.ReadN(int(length))
	if err != nil {
		return nil, err
	}

	return packet, nil
}

func (r *Transceiver) Write(msg encoding.BinaryMarshaler) error {
	packet, err := msg.MarshalBinary()
	if err != nil {
		return err
	}

	if _, err := r.stream.Write(packet); err != nil {
		return err
	}

	return nil
}

func (r *Transceiver) handleEcho(packet []byte) (ok bool, err error) {
	switch packet[0] {
	case openflow.OF10_VERSION:
		return r.handleOF10Echo(packet)
	case openflow.OF13_VERSION:
		return r.handleOF13Echo(packet)
	default:
		return false, openflow.ErrUnsupportedVersion
	}
}

func (r *Transceiver) handleOF10Echo(packet []byte) (ok bool, err error) {
	switch packet[1] {
	case of10.OFPT_ECHO_REQUEST:
		return true, r.handleEchoRequest(packet)
	case of10.OFPT_ECHO_REPLY:
		return true, r.handleEchoReply(packet)
	default:
		// Do not anything for other types of the message
		return false, nil
	}
}

func (r *Transceiver) handleOF13Echo(packet []byte) (handled bool, err error) {
	switch packet[1] {
	case of13.OFPT_ECHO_REQUEST:
		return true, r.handleEchoRequest(packet)
	case of13.OFPT_ECHO_REPLY:
		return true, r.handleEchoReply(packet)
	default:
		// Do not anything for other types of the message
		return false, nil
	}
}

func (r *Transceiver) dispatch(packet []byte) error {
	if packet[0] != r.version {
		return fmt.Errorf("mis-matched OpenFlow version: negotiated=%v, packet=%v", r.version, packet[0])
	}

	switch r.version {
	case openflow.OF10_VERSION:
		return r.handleOF10Message(packet)
	case openflow.OF13_VERSION:
		return r.handleOF13Message(packet)
	default:
		return openflow.ErrUnsupportedVersion
	}
}

func (r *Transceiver) handleOF10Message(packet []byte) error {
	switch packet[1] {
	case of10.OFPT_HELLO:
		return r.handleHello(packet)
	case of10.OFPT_ERROR:
		return r.handleError(packet)
	case of10.OFPT_FEATURES_REPLY:
		return r.handleFeaturesReply(packet)
	case of10.OFPT_GET_CONFIG_REPLY:
		return r.handleGetConfigReply(packet)
	case of10.OFPT_STATS_REPLY:
		switch binary.BigEndian.Uint16(packet[8:10]) {
		case of10.OFPST_DESC:
			return r.handleDescReply(packet)
		default:
			// Unsupported message. Do nothing.
			return nil
		}
	case of10.OFPT_PORT_STATUS:
		return r.handlePortStatus(packet)
	case of10.OFPT_FLOW_REMOVED:
		return r.handleFlowRemoved(packet)
	case of10.OFPT_PACKET_IN:
		return r.handlePacketIn(packet)
	default:
		// Unsupported message. Do nothing.
		return nil
	}
}

func (r *Transceiver) handleOF13Message(packet []byte) error {
	switch packet[1] {
	case of13.OFPT_HELLO:
		return r.handleHello(packet)
	case of13.OFPT_ERROR:
		return r.handleError(packet)
	case of13.OFPT_FEATURES_REPLY:
		return r.handleFeaturesReply(packet)
	case of13.OFPT_GET_CONFIG_REPLY:
		return r.handleGetConfigReply(packet)
	case of13.OFPT_MULTIPART_REPLY:
		switch binary.BigEndian.Uint16(packet[8:10]) {
		case of13.OFPMP_DESC:
			return r.handleDescReply(packet)
		case of13.OFPMP_PORT_DESC:
			return r.handlePortDescReply(packet)
		default:
			// Unsupported message. Do nothing.
			return nil
		}
	case of13.OFPT_PORT_STATUS:
		return r.handlePortStatus(packet)
	case of13.OFPT_FLOW_REMOVED:
		return r.handleFlowRemoved(packet)
	case of13.OFPT_PACKET_IN:
		return r.handlePacketIn(packet)
	default:
		// Unsupported message. Do nothing.
		return nil
	}
}

func (r *Transceiver) handleEchoRequest(packet []byte) error {
	msg, err := r.factory.NewEchoRequest()
	if err != nil {
		return err
	}
	if err := msg.UnmarshalBinary(packet); err != nil {
		return err
	}
	logger.Debug("received an ECHO_REQUEST packet")

	reply, err := r.factory.NewEchoReply()
	if err != nil {
		return err
	}
	// Copy transaction ID and data from the incoming echo request message
	reply.SetTransactionID(msg.TransactionID())
	reply.SetData(msg.Data())

	// Send the echo reply
	if err := r.Write(reply); err != nil {
		return errors.Wrap(err, "failed to send ECHO_REPLY message")
	}
	logger.Debug("sent an ECHO_REPLY packet")

	return nil
}

func (r *Transceiver) handleEchoReply(packet []byte) error {
	msg, err := r.factory.NewEchoReply()
	if err != nil {
		return err
	}
	if err := msg.UnmarshalBinary(packet); err != nil {
		return err
	}
	logger.Debug("received an ECHO_REPLY packet")

	data := msg.Data()
	if data == nil || len(data) != 8 {
		// I notice some broken switch sends an unexpected echo reply data.
		// So, ignores the soft error to avoid switch disconnection.
		logger.Debug("unexpected ECHO_REPLY data")
		return nil
	}
	timestamp := time.Time{}
	if err := timestamp.GobDecode(data); err != nil {
		logger.Debug("unexpected timestamp data in the ECHO_REPLY packet")
		return nil
	}

	// Network latency
	logger.Debugf("transceiver latency: %v", time.Now().Sub(timestamp))
	// Reset the ping counter
	r.pingCounter = 0

	return nil
}

func (r *Transceiver) handleHello(packet []byte) error {
	msg, err := r.factory.NewHello()
	if err != nil {
		return err
	}
	if err := msg.UnmarshalBinary(packet); err != nil {
		return err
	}

	return r.observer.OnHello(r.factory, r, msg)
}

func (r *Transceiver) handleError(packet []byte) error {
	msg, err := r.factory.NewError()
	if err != nil {
		return err
	}
	if err := msg.UnmarshalBinary(packet); err != nil {
		return err
	}

	return r.observer.OnError(r.factory, r, msg)
}

func (r *Transceiver) handleFeaturesReply(packet []byte) error {
	msg, err := r.factory.NewFeaturesReply()
	if err != nil {
		return err
	}
	if err := msg.UnmarshalBinary(packet); err != nil {
		return err
	}

	return r.observer.OnFeaturesReply(r.factory, r, msg)
}

func (r *Transceiver) handleGetConfigReply(packet []byte) error {
	msg, err := r.factory.NewGetConfigReply()
	if err != nil {
		return err
	}
	if err := msg.UnmarshalBinary(packet); err != nil {
		return err
	}

	return r.observer.OnGetConfigReply(r.factory, r, msg)
}

func (r *Transceiver) handleDescReply(packet []byte) error {
	msg, err := r.factory.NewDescReply()
	if err != nil {
		return err
	}
	if err := msg.UnmarshalBinary(packet); err != nil {
		return err
	}

	return r.observer.OnDescReply(r.factory, r, msg)
}

func (r *Transceiver) handlePortDescReply(packet []byte) error {
	msg, err := r.factory.NewPortDescReply()
	if err != nil {
		return err
	}
	if err := msg.UnmarshalBinary(packet); err != nil {
		return err
	}

	return r.observer.OnPortDescReply(r.factory, r, msg)
}

func (r *Transceiver) handlePortStatus(packet []byte) error {
	msg, err := r.factory.NewPortStatus()
	if err != nil {
		return err
	}
	if err := msg.UnmarshalBinary(packet); err != nil {
		return err
	}

	return r.observer.OnPortStatus(r.factory, r, msg)
}

func (r *Transceiver) handleFlowRemoved(packet []byte) error {
	msg, err := r.factory.NewFlowRemoved()
	if err != nil {
		return err
	}
	if err := msg.UnmarshalBinary(packet); err != nil {
		return err
	}

	return r.observer.OnFlowRemoved(r.factory, r, msg)
}

func (r *Transceiver) handlePacketIn(packet []byte) error {
	msg, err := r.factory.NewPacketIn()
	if err != nil {
		return err
	}
	if err := msg.UnmarshalBinary(packet); err != nil {
		return err
	}

	return r.observer.OnPacketIn(r.factory, r, msg)
}

func (r *Transceiver) Close() error {
	if r.closed {
		return nil
	}

	if err := r.stream.Close(); err != nil {
		return err
	}
	r.closed = true

	return nil
}
