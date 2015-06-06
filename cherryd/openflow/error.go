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

type Error interface {
	Header
	Class() uint16 // Error type
	Code() uint16
	Data() []byte
	encoding.BinaryUnmarshaler
}

type BaseError struct {
	Message
	class uint16
	code  uint16
	data  []byte
}

func (r BaseError) Class() uint16 {
	return r.class
}

func (r BaseError) Code() uint16 {
	return r.code
}

func (r BaseError) Data() []byte {
	return r.data
}

func (r *BaseError) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 4 {
		return ErrInvalidPacketLength
	}
	r.class = binary.BigEndian.Uint16(payload[0:2])
	r.code = binary.BigEndian.Uint16(payload[2:4])
	if len(payload) > 4 {
		r.data = payload[4:]
	}

	return nil
}
