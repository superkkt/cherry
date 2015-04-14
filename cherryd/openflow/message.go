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
)

var messageParser map[uint8]func([]byte) (Incoming, error)

func init() {
	messageParser = make(map[uint8]func([]byte) (Incoming, error))
}

type Message struct {
	version uint8
	msgType uint8
	xid     uint32
	length  uint16
	payload []byte
}

func NewMessage(version uint8, msgType uint8, xid uint32) Message {
	return Message{
		version: version,
		msgType: msgType,
		xid:     xid,
		length:  8,
	}
}

func (r *Message) Version() uint8 {
	return r.version
}

func (r *Message) Type() uint8 {
	return r.msgType
}

func (r *Message) TransactionID() uint32 {
	return r.xid
}

func (r *Message) SetPayload(payload []byte) {
	r.payload = payload
	if payload == nil {
		r.length = 8
	} else {
		r.length = uint16(8 + len(payload))
	}
}

func (r *Message) Payload() []byte {
	if r.payload == nil {
		return nil
	}

	v := make([]byte, len(r.payload))
	copy(v, r.payload)

	return v
}

func (r *Message) MarshalBinary() ([]byte, error) {
	var length uint16 = 8
	if r.payload != nil {
		length += uint16(len(r.payload))
	}

	v := make([]byte, length)
	v[0] = r.version
	v[1] = r.msgType
	binary.BigEndian.PutUint16(v[2:4], length)
	binary.BigEndian.PutUint32(v[4:8], r.xid)
	if length > 8 {
		copy(v[8:], r.payload)
	}

	return v, nil
}

func (r *Message) UnmarshalBinary(data []byte) error {
	if data == nil || len(data) < 8 {
		return ErrInvalidPacketLength
	}

	r.version = data[0]
	r.msgType = data[1]
	r.length = binary.BigEndian.Uint16(data[2:4])
	if r.length < 8 || len(data) < int(r.length) {
		return ErrInvalidPacketLength
	}
	r.xid = binary.BigEndian.Uint32(data[4:8])
	r.payload = data[8:r.length]

	return nil
}

type Header interface {
	Version() uint8
	Type() uint8
	TransactionID() uint32
}

type Outgoing interface {
	Header
	encoding.BinaryMarshaler
}

type Incoming interface {
	Header
	encoding.BinaryUnmarshaler
}

func RegisterParser(version uint8, parser func([]byte) (Incoming, error)) {
	if parser == nil {
		panic("nil message parser function")
	}
	messageParser[version] = parser
}

// TODO: set deadline before passing conn to this function
func ReadMessage(stream *Stream) (Incoming, error) {
	header, err := stream.Peek(8) // peek ofp_header
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint16(header[2:4])
	if length < 8 {
		return nil, ErrInvalidPacketLength
	}
	packet, err := stream.ReadN(int(length))
	if err != nil {
		return nil, err
	}

	var msg Incoming
	switch packet[1] {
	// OFPT_HELLO
	case 0x0:
		msg = new(Hello)
	// OFPT_ERROR
	case 0x1:
		msg = new(Error)
	// OFPT_ECHO_REQUEST
	case 0x2:
		msg = new(EchoRequest)
	// OFPT_ECHO_REPLY
	case 0x3:
		msg = new(EchoReply)
	// All other message types
	default:
		// Find a message parser for the message version
		parser, ok := messageParser[packet[0]]
		if !ok {
			return nil, ErrUnsupportedVersion
		}
		return parser(packet)
	}

	if err := msg.UnmarshalBinary(packet); err != nil {
		return nil, err
	}
	return msg, nil
}

// TODO: set deadline before passing conn to this function
func WriteMessage(stream *Stream, msg Outgoing) error {
	v, err := msg.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = stream.Write(v)
	if err != nil {
		return err
	}

	return nil
}
