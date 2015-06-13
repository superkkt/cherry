/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package trans

import (
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"git.sds.co.kr/cherry.git/cherryd/openflow/of10"
	"git.sds.co.kr/cherry.git/cherryd/openflow/of13"
	"io"
	"time"
)

const (
	// Allowed idle time before we send an echo request to a switch (in seconds)
	MaxIdleTime = 30
)

type Transceiver struct {
	stream    *Stream
	observer  Handler
	version   uint8
	factory   openflow.Factory
	timestamp time.Time     // Last activated time
	latency   time.Duration // Network latency measured by echo request and reply
}

type Handler interface {
	OnHello(openflow.Factory, io.Writer, openflow.Hello) error
	OnError(openflow.Factory, io.Writer, openflow.Error) error
	OnFeaturesReply(openflow.Factory, io.Writer, openflow.FeaturesReply) error
	OnGetConfigReply(openflow.Factory, io.Writer, openflow.GetConfigReply) error
	OnDescReply(openflow.Factory, io.Writer, openflow.DescReply) error
	OnPortDescReply(openflow.Factory, io.Writer, openflow.PortDescReply) error
	OnPortStatus(openflow.Factory, io.Writer, openflow.PortStatus) error
	OnFlowRemoved(openflow.Factory, io.Writer, openflow.FlowRemoved) error
	OnPacketIn(openflow.Factory, io.Writer, openflow.PacketIn) error
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

func (r *Transceiver) Latency() time.Duration {
	return r.latency
}

func (r *Transceiver) negotiate(packet []byte) error {
	// The first message should be HELLO
	if packet[1] != 0x00 {
		return errors.New("negotiation error: missing HELLO message")
	}

	// Version negotiation
	if packet[0] < openflow.OF13_VERSION {
		r.version = openflow.OF10_VERSION
		r.factory = of10.NewFactory()
	} else {
		r.version = openflow.OF13_VERSION
		r.factory = of13.NewFactory()
	}

	return nil
}

func (r *Transceiver) updateTimestamp() {
	r.timestamp = time.Now()
}

func (r *Transceiver) ping() error {
	// Max idle time is exceeded?
	if time.Now().Before(r.timestamp.Add(MaxIdleTime * time.Second)) {
		return nil
	}
	return r.sendEchoRequest()
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
	echo, err := r.factory.NewEchoRequest()
	if err != nil {
		return err
	}
	// We use current timestamp to check network latency between our controller and a switch.
	timestamp, err := time.Now().GobEncode()
	if err != nil {
		return err
	}
	if err := echo.SetData(timestamp); err != nil {
		return err
	}
	if err := r.writePacket(echo); err != nil {
		return fmt.Errorf("failed to send ECHO_REQUEST message: %v", err)
	}

	return nil
}

// TODO: Use context to shutdown a running transceiver
func (r *Transceiver) Run() error {
	// Read initial packet
	packet, err := r.readPacket()
	if err != nil {
		return err
	}
	if err := r.negotiate(packet); err != nil {
		return err
	}

	// Infinite loop
	for {
		if err := r.dispatch(packet); err != nil {
			return err
		}
		r.updateTimestamp()

	retry:
		// Read next packet
		packet, err = r.readPacket()
		if err == nil {
			// Go to dispatch the next packet
			continue
		}
		// Ignore timeout error
		if !isTimeout(err) {
			return err
		}
		if err := r.ping(); err != nil {
			return err
		}
		// Read again
		goto retry
	}

	// Never reached
	return nil
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

func (r *Transceiver) writePacket(msg encoding.BinaryMarshaler) error {
	packet, err := msg.MarshalBinary()
	if err != nil {
		return err
	}
	if _, err := r.stream.Write(packet); err != nil {
		return err
	}

	return nil
}

func (r *Transceiver) dispatch(packet []byte) error {
	if packet[0] != r.version {
		m := fmt.Sprintf("mis-matched OpenFlow version: negotiated=%v, packet=%v", r.version, packet[0])
		return errors.New(m)
	}

	var msg interface{}
	var err error

	switch r.version {
	case openflow.OF10_VERSION:
		msg, err = r.parseOF10Message(packet)
	case openflow.OF13_VERSION:
		msg, err = r.parseOF13Message(packet)
	default:
		return openflow.ErrUnsupportedVersion
	}
	if err != nil {
		return err
	}

	return r.callback(msg)
}

func (r *Transceiver) parseOF10Message(packet []byte) (interface{}, error) {
	var msg encoding.BinaryUnmarshaler
	var err error

	switch packet[1] {
	case of10.OFPT_HELLO:
		msg, err = r.factory.NewHello()
	case of10.OFPT_ERROR:
		msg, err = r.factory.NewError()
	case of10.OFPT_ECHO_REQUEST:
		msg, err = r.factory.NewEchoRequest()
	case of10.OFPT_ECHO_REPLY:
		msg, err = r.factory.NewEchoReply()
	case of10.OFPT_FEATURES_REPLY:
		msg, err = r.factory.NewFeaturesReply()
	case of10.OFPT_GET_CONFIG_REPLY:
		msg, err = r.factory.NewGetConfigReply()
	case of10.OFPT_STATS_REPLY:
		switch binary.BigEndian.Uint16(packet[8:10]) {
		case of10.OFPST_DESC:
			msg, err = r.factory.NewDescReply()
		default:
			return nil, openflow.ErrUnsupportedMessage
		}
	case of10.OFPT_PORT_STATUS:
		msg, err = r.factory.NewPortStatus()
	case of10.OFPT_FLOW_REMOVED:
		msg, err = r.factory.NewFlowRemoved()
	case of10.OFPT_PACKET_IN:
		msg, err = r.factory.NewPacketIn()
	default:
		return nil, openflow.ErrUnsupportedMessage
	}

	if err != nil {
		return nil, err
	}
	if err := msg.UnmarshalBinary(packet); err != nil {
		return nil, err
	}

	return msg, nil
}

func (r *Transceiver) parseOF13Message(packet []byte) (interface{}, error) {
	var msg encoding.BinaryUnmarshaler
	var err error

	switch packet[1] {
	case of13.OFPT_HELLO:
		msg, err = r.factory.NewHello()
	case of13.OFPT_ERROR:
		msg, err = r.factory.NewError()
	case of13.OFPT_ECHO_REQUEST:
		msg, err = r.factory.NewEchoRequest()
	case of13.OFPT_ECHO_REPLY:
		msg, err = r.factory.NewEchoReply()
	case of13.OFPT_FEATURES_REPLY:
		msg, err = r.factory.NewFeaturesReply()
	case of13.OFPT_GET_CONFIG_REPLY:
		msg, err = r.factory.NewGetConfigReply()
	case of13.OFPT_MULTIPART_REPLY:
		switch binary.BigEndian.Uint16(packet[8:10]) {
		case of13.OFPMP_DESC:
			msg, err = r.factory.NewDescReply()
		case of13.OFPMP_PORT_DESC:
			msg, err = r.factory.NewPortDescReply()
		default:
			return nil, openflow.ErrUnsupportedMessage
		}
	case of13.OFPT_PORT_STATUS:
		msg, err = r.factory.NewPortStatus()
	case of13.OFPT_FLOW_REMOVED:
		msg, err = r.factory.NewFlowRemoved()
	case of13.OFPT_PACKET_IN:
		msg, err = r.factory.NewPacketIn()
	default:
		return nil, openflow.ErrUnsupportedMessage
	}

	if err != nil {
		return nil, err
	}
	if err := msg.UnmarshalBinary(packet); err != nil {
		return nil, err
	}

	return msg, nil
}

func (r *Transceiver) callback(msg interface{}) error {
	switch v := msg.(type) {
	// Echo request and reply are only processed in this transceiver
	case openflow.EchoRequest:
		return r.handleEchoRequest(v)
	case openflow.EchoReply:
		return r.handleEchoReply(v)
	case openflow.Hello:
		return r.observer.OnHello(r.factory, r.stream, v)
	case openflow.Error:
		return r.observer.OnError(r.factory, r.stream, v)
	case openflow.FeaturesReply:
		return r.observer.OnFeaturesReply(r.factory, r.stream, v)
	case openflow.GetConfigReply:
		return r.observer.OnGetConfigReply(r.factory, r.stream, v)
	case openflow.DescReply:
		return r.observer.OnDescReply(r.factory, r.stream, v)
	case openflow.PortDescReply:
		return r.observer.OnPortDescReply(r.factory, r.stream, v)
	case openflow.PortStatus:
		return r.observer.OnPortStatus(r.factory, r.stream, v)
	case openflow.FlowRemoved:
		return r.observer.OnFlowRemoved(r.factory, r.stream, v)
	case openflow.PacketIn:
		return r.observer.OnPacketIn(r.factory, r.stream, v)
	default:
		panic("unexpected message type")
	}
}

func (r *Transceiver) handleEchoRequest(msg openflow.EchoRequest) error {
	// Send echo reply
	reply, err := r.factory.NewEchoReply()
	if err != nil {
		return err
	}
	// Copy transaction ID and data from the incoming echo request message
	if err := reply.SetTransactionID(msg.TransactionID()); err != nil {
		return err
	}
	if err := reply.SetData(msg.Data()); err != nil {
		return err
	}
	if err := r.writePacket(reply); err != nil {
		return fmt.Errorf("failed to send ECHO_REPLY message: %v", err)
	}

	return nil
}

func (r *Transceiver) handleEchoReply(msg openflow.EchoReply) error {
	data := msg.Data()
	if data == nil || len(data) != 8 {
		return errors.New("unexpected ECHO_REPLY data")
	}
	timestamp := time.Time{}
	if err := timestamp.GobDecode(data); err != nil {
		return err
	}
	// Update network latency
	r.latency = time.Now().Sub(timestamp)

	return nil
}
