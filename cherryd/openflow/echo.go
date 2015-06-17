/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding"
)

type Echo interface {
	Data() []byte
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	// Error() returns last error message
	Error() error
	Header
	SetData(data []byte)
}

type EchoRequest interface {
	Echo
}

type EchoReply interface {
	Echo
}

type BaseEcho struct {
	err error
	Message
	data []byte
}

func (r *BaseEcho) Data() []byte {
	return r.data
}

func (r *BaseEcho) SetData(data []byte) {
	if data == nil {
		panic("data is nil")
	}
	r.data = data
}

func (r *BaseEcho) Error() error {
	return r.err
}

func (r *BaseEcho) MarshalBinary() ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}

	r.SetPayload(r.data)
	return r.Message.MarshalBinary()
}

func (r *BaseEcho) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}
	r.data = r.Payload()

	return nil
}
