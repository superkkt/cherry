/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding"
	"errors"
)

type Echo interface {
	Header
	Data() []byte
	SetData(data []byte) error
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type EchoRequest interface {
	Echo
}

type EchoReply interface {
	Echo
}

type BaseEcho struct {
	Message
	data []byte
}

func (r *BaseEcho) Data() []byte {
	return r.data
}

func (r *BaseEcho) SetData(data []byte) error {
	if data == nil {
		return errors.New("data is nil")
	}
	r.data = data
	return nil
}

func (r *BaseEcho) MarshalBinary() ([]byte, error) {
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
