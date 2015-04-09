/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

import (
	"encoding/binary"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type Error struct {
	header openflow.Header
	Type   uint16
	Code   uint16
	Data   []byte
}

func (r *Error) Header() openflow.Header {
	return r.header
}

func (r *Error) MarshalBinary() ([]byte, error) {
	return nil, openflow.ErrUnsupportedMarshaling
}

func (r *Error) UnmarshalBinary(data []byte) error {
	if err := r.header.UnmarshalBinary(data); err != nil {
		return err
	}
	if r.header.Length < 12 || len(data) < int(r.header.Length) {
		return openflow.ErrInvalidPacketLength
	}

	r.Type = binary.BigEndian.Uint16(data[8:10])
	r.Code = binary.BigEndian.Uint16(data[10:12])
	if r.header.Length > 12 {
		length := r.header.Length - 12
		r.Data = data[12 : 12+length]
	}

	return nil
}
