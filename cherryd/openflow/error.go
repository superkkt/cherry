/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding/binary"
	"errors"
)

var (
	ErrInvalidPacketLength     = errors.New("invalid packet length")
	ErrUnsupportedVersion      = errors.New("unsupported protocol version")
	ErrUnsupportedMarshaling   = errors.New("invalid marshaling")
	ErrUnsupportedUnmarshaling = errors.New("invalid unmarshaling")
	ErrUnsupportedMessage      = errors.New("unsupported message type")
)

func IsTimeout(err error) bool {
	type Timeout interface {
		Timeout() bool
	}

	if v, ok := err.(Timeout); ok {
		return v.Timeout()
	}

	return false
}

type Error struct {
	header Header
	Type   uint16
	Code   uint16
	Data   []byte
}

func (r *Error) Header() Header {
	return r.header
}

func (r *Error) MarshalBinary() ([]byte, error) {
	return nil, ErrUnsupportedMarshaling
}

func (r *Error) UnmarshalBinary(data []byte) error {
	if err := r.header.UnmarshalBinary(data); err != nil {
		return err
	}
	if r.header.Length < 12 || len(data) < int(r.header.Length) {
		return ErrInvalidPacketLength
	}

	r.Type = binary.BigEndian.Uint16(data[8:10])
	r.Code = binary.BigEndian.Uint16(data[10:12])
	if r.header.Length > 12 {
		length := r.header.Length - 12
		r.Data = data[12 : 12+length]
	}

	return nil
}
