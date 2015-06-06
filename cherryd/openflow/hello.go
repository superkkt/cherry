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

type Hello interface {
	Header
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type BaseHello struct {
	Message
}

func (r *BaseHello) MarshalBinary() ([]byte, error) {
	return r.Message.MarshalBinary()
}

func (r *BaseHello) UnmarshalBinary(data []byte) error {
	return r.Message.UnmarshalBinary(data)
}
