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
	"reflect"
	"time"
)

const (
	// Allowed idle time before we send an echo request to a switch (in seconds)
	MaxIdleTime = 30
)

type Writer interface {
	Write(msg encoding.BinaryMarshaler) error
}

type Transceiver struct {
	stream    *Stream
	observer  Handler
	version   uint8
	factory   openflow.Factory
	timestamp time.Time     // Last activated time
	latency   time.Duration // Network latency measured by echo request and reply
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
		// XXX: debugging
		fmt.Printf("Negotiated as OF10\n")
	} else {
		r.version = openflow.OF13_VERSION
		r.factory = of13.NewFactory()
		fmt.Printf("Negotiated as OF13\n")
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
	if err := r.Write(echo); err != nil {
		return fmt.Errorf("failed to send ECHO_REQUEST message: %v", err)
	}

	return nil
}

// TODO: Use context to shutdown a running transceiver
func (r *Transceiver) Run() error {
	// XXX: deubgging
	fmt.Printf("Reading initial packet..\n")

	// Read initial packet
	packet, err := r.readPacket()
	if err != nil {
		return err
	}

	// XXX: deubgging
	fmt.Printf("Negotiating..\n")

	if err := r.negotiate(packet); err != nil {
		return err
	}

	// Infinite loop
	for {
		// XXX: deubgging
		fmt.Printf("Dispatching..\n")

		if err := r.dispatch(packet); err != nil {
			return err
		}
		// XXX: deubgging
		fmt.Printf("Dispatch Done..\n")
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
	// XXX: deubgging
	fmt.Printf("Peeking..\n")
	header, err := r.stream.Peek(8) // peek ofp_header
	if err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint16(header[2:4])
	// XXX: deubgging
	fmt.Printf("Packet length=%v\n", length)
	if length < 8 {
		return nil, openflow.ErrInvalidPacketLength
	}
	// XXX: deubgging
	fmt.Printf("Reading %v bytes..\n", length)
	packet, err := r.stream.ReadN(int(length))
	if err != nil {
		return nil, err
	}

	return packet, nil
}

func (r *Transceiver) Write(msg encoding.BinaryMarshaler) error {
	// XXX: deubgging
	fmt.Printf("Write() is called..\n")

	packet, err := msg.MarshalBinary()
	if err != nil {
		return err
	}

	// XXX: deubgging
	fmt.Printf("Writing a message.. type=%v\n", reflect.TypeOf(msg))
	if _, err := r.stream.Write(packet); err != nil {
		return err
	}
	// XXX: deubgging
	fmt.Printf("Writing is done\n")

	return nil
}

func (r *Transceiver) dispatch(packet []byte) error {
	if packet[0] != r.version {
		m := fmt.Sprintf("mis-matched OpenFlow version: negotiated=%v, packet=%v", r.version, packet[0])
		return errors.New(m)
	}

	switch r.version {
	case openflow.OF10_VERSION:
		return r.parseOF10Message(packet)
	case openflow.OF13_VERSION:
		return r.parseOF13Message(packet)
	default:
		return openflow.ErrUnsupportedVersion
	}
}

func (r *Transceiver) parseOF10Message(packet []byte) error {
	// XXX: deubgging
	fmt.Printf("packet[1] is %v\n", packet[1])

	switch packet[1] {
	case of10.OFPT_HELLO:
		return r.handleHello(packet)
	case of10.OFPT_ERROR:
		return r.handleError(packet)
	case of10.OFPT_ECHO_REQUEST:
		return r.handleEchoRequest(packet)
	case of10.OFPT_ECHO_REPLY:
		return r.handleEchoReply(packet)
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

func (r *Transceiver) parseOF13Message(packet []byte) error {
	// XXX: deubgging
	fmt.Printf("packet[1] is %v\n", packet[1])

	switch packet[1] {
	case of13.OFPT_HELLO:
		// XXX: deubgging
		fmt.Printf("NewHello..\n")
		return r.handleHello(packet)
	case of13.OFPT_ERROR:
		// XXX: deubgging
		fmt.Printf("NewError..\n")
		return r.handleError(packet)
	case of13.OFPT_ECHO_REQUEST:
		// XXX: deubgging
		fmt.Printf("NewEchoRequest..\n")
		return r.handleEchoRequest(packet)
	case of13.OFPT_ECHO_REPLY:
		// XXX: deubgging
		fmt.Printf("NewEchoReply..\n")
		return r.handleEchoReply(packet)
	case of13.OFPT_FEATURES_REPLY:
		// XXX: deubgging
		fmt.Printf("NewFeaturesReply..\n")
		return r.handleFeaturesReply(packet)
	case of13.OFPT_GET_CONFIG_REPLY:
		// XXX: deubgging
		fmt.Printf("NewGetConfigReply..\n")
		return r.handleGetConfigReply(packet)
	case of13.OFPT_MULTIPART_REPLY:
		switch binary.BigEndian.Uint16(packet[8:10]) {
		case of13.OFPMP_DESC:
			// XXX: deubgging
			fmt.Printf("NewDescReply..\n")
			return r.handleDescReply(packet)
		case of13.OFPMP_PORT_DESC:
			// XXX: deubgging
			fmt.Printf("NewPortDescReply..\n")
			return r.handlePortDescReply(packet)
		default:
			// Unsupported message. Do nothing.
			return nil
		}
	case of13.OFPT_PORT_STATUS:
		// XXX: deubgging
		fmt.Printf("NewPortStatus..\n")
		return r.handlePortStatus(packet)
	case of13.OFPT_FLOW_REMOVED:
		// XXX: deubgging
		fmt.Printf("NewFlowRemoved..\n")
		return r.handleFlowRemoved(packet)
	case of13.OFPT_PACKET_IN:
		// XXX: deubgging
		fmt.Printf("NewPacketIn..\n")
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
	if err := r.Write(reply); err != nil {
		return fmt.Errorf("failed to send ECHO_REPLY message: %v", err)
	}

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
	return r.stream.Close()
}
