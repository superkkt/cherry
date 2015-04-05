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
)

var (
	ErrInvalidPacketLength = errors.New("invalid packet length")
	ErrUnsupportedVersion  = errors.New("unsupported protocol version")
	ErrInvalidMarshaling   = errors.New("invalid marshaling")
	ErrInvalidUnmarshaling = errors.New("invalid unmarshaling")
	ErrUnsupportedMessage  = errors.New("unsupported message type")
)

var messageParser map[uint8]func([]byte) (Message, error)

func init() {
	messageParser = make(map[uint8]func([]byte) (Message, error))
}

type Header struct {
	Version uint8
	Type    uint8
	Length  uint16
	XID     uint32
}

func (r *Header) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	v[0] = r.Version
	v[1] = r.Type
	binary.BigEndian.PutUint16(v[2:4], r.Length)
	binary.BigEndian.PutUint32(v[4:8], r.XID)

	return v, nil
}

func (r *Header) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return ErrInvalidPacketLength
	}

	r.Version = data[0]
	r.Type = data[1]
	r.Length = binary.BigEndian.Uint16(data[2:4])
	if r.Length < 8 {
		return ErrInvalidPacketLength
	}
	r.XID = binary.BigEndian.Uint32(data[4:8])

	return nil
}

type Message interface {
	Header() Header
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

func RegisterParser(version uint8, parser func([]byte) (Message, error)) {
	if parser == nil {
		panic("nil message parser function")
	}
	messageParser[version] = parser
}

// TODO: set deadline before passing conn to this function
func ReadMessage(stream *Stream) (Message, error) {
	header, err := stream.peek(8) // peek ofp_header
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint16(header[2:4])
	if length < 8 {
		return nil, ErrInvalidPacketLength
	}
	packet, err := stream.readN(int(length))
	if err != nil {
		return nil, err
	}

	// OFPT_HELLO
	if packet[1] == 0 {
		v := new(Hello)
		if err := v.UnmarshalBinary(packet); err != nil {
			return nil, err
		}
		return v, nil
	}

	parser, ok := messageParser[packet[0]]
	if !ok {
		return nil, ErrUnsupportedVersion
	}

	return parser(packet)
}

// TODO: set deadline before passing conn to this function
func WriteMessage(stream *Stream, msg Message) error {
	v, err := msg.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = stream.write(v)
	if err != nil {
		return err
	}

	return nil
}
