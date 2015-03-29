/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding"
	"encoding/binary"
	"errors"
	"git.sds.co.kr/bosomi.git/socket"
	"golang.org/x/net/context"
	"log"
	"time"
)

var (
	ErrUnsupportedMsgType  = errors.New("unsupported message type")
	ErrNoNegotiated        = errors.New("no negotiated session")
	ErrInvalidPacketLength = errors.New("invalid packet length")
)

// The connection will be disconnected if a handler function returns error.
type MessageHandler struct {
	HelloMessage         func(*HelloMessage) error
	ErrorMessage         func(*ErrorMessage) error
	FeaturesReplyMessage func(*FeaturesReplyMessage) error
	EchoRequestMessage   func(*EchoRequestMessage) error
	EchoReplyMessage     func(*EchoReplyMessage) error
	PortStatusMessage    func(*PortStatusMessage) error
	PacketInMessage      func(*PacketInMessage) error
	FlowRemovedMessage   func(*FlowRemovedMessage) error
}

type Config struct {
	Log          *log.Logger
	Socket       *socket.Conn
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Handlers     MessageHandler
}

type Transceiver struct {
	Config
	xid        uint32 // Transaction ID associated with OpenFlow packets
	negotiated bool   // Negotiation of protocol version
}

func NewTransceiver(config Config) (*Transceiver, error) {
	if config.Log == nil {
		return nil, errors.New("Config.Log is essential parameter")
	}
	if config.Socket == nil {
		return nil, errors.New("Config.Socket is essential parameter")
	}

	return &Transceiver{
		Config: config,
	}, nil
}

func (r *Transceiver) send(data encoding.BinaryMarshaler) error {
	v, err := data.MarshalBinary()
	if err != nil {
		return err
	}

	if r.WriteTimeout > 0 {
		r.Socket.SetDeadline(time.Now().Add(r.WriteTimeout))
		defer r.Socket.SetDeadline(time.Time{})
	}
	_, err = r.Socket.Write(v)
	if err != nil {
		return err
	}

	return nil
}

func (r *Transceiver) getTransactionID() uint32 {
	v := r.xid
	r.xid++
	return v
}

func (r *Transceiver) SendFlowModifyMessage(msg *FlowModifyMessage) error {
	msg.header = Header{
		Version: 0x01, // OF1.0
		Type:    OFPT_FLOW_MOD,
		Xid:     r.getTransactionID(),
	}
	return r.send(msg)
}

func (r *Transceiver) sendHelloMessage() error {
	msg := &HelloMessage{
		Header{
			Version: 0x01, // OF1.0
			Type:    OFPT_HELLO,
			Xid:     r.getTransactionID(),
		},
	}

	return r.send(msg)
}

func (r *Transceiver) sendFeaturesRequestMessage() error {
	msg := &FeaturesRequestMessage{
		Header{
			Version: 0x01, // OF1.0
			Type:    OFPT_FEATURES_REQUEST,
			Xid:     r.getTransactionID(),
		},
	}

	return r.send(msg)
}

func (r *Transceiver) sendNegotiationFailedMessage(data string) error {
	msg := &ErrorMessage{
		Header: Header{
			Version: 0x01, // OF1.0
			Type:    OFPT_ERROR,
			Xid:     r.getTransactionID(),
		},
		Type: OFPET_HELLO_FAILED,
		Code: OFPHFC_INCOMPATIBLE,
		Data: []byte(data),
	}

	return r.send(msg)
}

func parsePacket(packet []byte) (interface{}, error) {
	var msg encoding.BinaryUnmarshaler

	switch PacketType(packet[1]) {
	case OFPT_HELLO:
		msg = &HelloMessage{}
	case OFPT_ERROR:
		msg = &ErrorMessage{}
	case OFPT_FEATURES_REPLY:
		msg = &FeaturesReplyMessage{}
	case OFPT_ECHO_REQUEST:
		msg = &EchoRequestMessage{}
	case OFPT_ECHO_REPLY:
		msg = &EchoReplyMessage{}
	case OFPT_PORT_STATUS:
		msg = &PortStatusMessage{}
	case OFPT_PACKET_IN:
		msg = &PacketInMessage{}
	case OFPT_FLOW_REMOVED:
		msg = &FlowRemovedMessage{}
	default:
		return nil, ErrUnsupportedMsgType
	}

	if err := msg.UnmarshalBinary(packet); err != nil {
		return nil, err
	}
	return msg, nil
}

func (r *Transceiver) handleMessage(ctx context.Context, msg interface{}) error {
	if _, ok := msg.(*HelloMessage); !ok && !r.negotiated {
		return ErrNoNegotiated
	}

	// FIXME: Try to use reflection to remove these manual callback calls
	switch v := msg.(type) {
	case *HelloMessage:
		if r.Handlers.HelloMessage != nil {
			err := r.Handlers.HelloMessage(v)
			if err != nil {
				r.sendNegotiationFailedMessage(err.Error())
				return err
			}

			// TODO: Set to send whole packet data in PACKET_IN using max_len(?)

			if err := r.sendFeaturesRequestMessage(); err != nil {
				return err
			}
			go r.pinger(ctx)
			r.negotiated = true
		}

	case *ErrorMessage:
		if r.Handlers.ErrorMessage != nil {
			return r.Handlers.ErrorMessage(v)
		}

	case *FeaturesReplyMessage:
		if r.Handlers.FeaturesReplyMessage != nil {
			return r.Handlers.FeaturesReplyMessage(v)
		}

	case *EchoRequestMessage:
		if r.Handlers.EchoRequestMessage != nil {
			if err := r.Handlers.EchoRequestMessage(v); err != nil {
				return err
			}
			if err := r.sendEchoReply(v.Data); err != nil {
			}
		}

	case *EchoReplyMessage:
		if r.Handlers.EchoReplyMessage != nil {
			return r.Handlers.EchoReplyMessage(v)
		}

	case *PortStatusMessage:
		if r.Handlers.PortStatusMessage != nil {
			return r.Handlers.PortStatusMessage(v)
		}

	case *PacketInMessage:
		if r.Handlers.PacketInMessage != nil {
			return r.Handlers.PacketInMessage(v)
		}

	case *FlowRemovedMessage:
		if r.Handlers.FlowRemovedMessage != nil {
			return r.Handlers.FlowRemovedMessage(v)
		}

	default:
		panic("unsupported message type!")
	}

	return nil
}

func (r *Transceiver) readMessage() (interface{}, error) {
	if r.ReadTimeout > 0 {
		r.Socket.SetDeadline(time.Now().Add(r.ReadTimeout))
		defer r.Socket.SetDeadline(time.Time{})
	}

	header, err := r.Socket.Peek(8) // peek ofp_header
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint16(header[2:4])
	if length < 8 {
		return nil, ErrInvalidPacketLength
	}

	packet, err := r.Socket.ReadN(int(length))
	if err != nil {
		return nil, err
	}

	return parsePacket(packet)
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
	msg := &EchoRequestMessage{
		EchoMessage{
			Header: Header{
				Version: 0x01, // OF1.0
				Type:    OFPT_ECHO_REQUEST,
				Xid:     r.getTransactionID(),
			},
			Data: make([]byte, 8),
		},
	}
	timestamp := time.Now().Unix()
	binary.BigEndian.PutUint64(msg.Data, uint64(timestamp))

	return r.send(msg)
}

func (r *Transceiver) sendEchoReply(data []byte) error {
	msg := &EchoRequestMessage{
		EchoMessage{
			Header: Header{
				Version: 0x01, // OF1.0
				Type:    OFPT_ECHO_REPLY,
				Xid:     r.getTransactionID(),
			},
		},
	}

	if data != nil && len(data) > 0 {
		msg.Data = make([]byte, len(data))
		copy(msg.Data, data)
	}

	return r.send(msg)
}

func (r *Transceiver) pinger(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			r.sendEchoRequest()

		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (r *Transceiver) Run(ctx context.Context) {
	defer r.Socket.Close()

	if err := r.sendHelloMessage(); err != nil {
		r.Log.Print(err)
		return
	}

	// Reader goroutine
	receivedMsg := make(chan interface{})
	go func() {
		for {
			msg, err := r.readMessage()
			if err != nil {
				switch {
				case isTimeout(err):
					// Ignore timeout error
					continue

				case err == ErrUnsupportedMsgType:
					r.Log.Print(err)
					continue

				default:
					r.Log.Print(err)
					close(receivedMsg)
					return
				}
			}
			receivedMsg <- msg
		}
	}()

	for {
		select {
		case msg, ok := <-receivedMsg:
			if !ok {
				return
			}
			if err := r.handleMessage(ctx, msg); err != nil {
				r.Log.Print(err)
				// Reader goroutine will be finished after it detects socket disconnection
				return
			}

		// TODO: add this manager to the device pool with DPID

		case <-ctx.Done():
			return
		}
	}
}
