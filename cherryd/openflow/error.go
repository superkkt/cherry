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
	ErrInvalidPacketLength = errors.New("invalid packet length")
	ErrUnsupportedVersion  = errors.New("unsupported protocol version")
	ErrUnsupportedMessage  = errors.New("unsupported message type")
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
	Message
	Class uint16 // Error type
	Code  uint16
	Data  []byte
}

func (r *Error) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 4 {
		return ErrInvalidPacketLength
	}
	r.Class = binary.BigEndian.Uint16(payload[0:2])
	r.Code = binary.BigEndian.Uint16(payload[2:4])
	if len(payload) > 4 {
		r.Data = payload[4:]
	}

	return nil
}
