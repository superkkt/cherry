/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding/binary"
)

type ErrorMessage struct {
	Header
	Type uint16
	Code uint16
	Data []byte
}

func (r *ErrorMessage) MarshalBinary() ([]byte, error) {
	header, err := r.Header.MarshalBinary()
	if err != nil {
		return nil, err
	}

	data := make([]byte, r.Length)
	copy(data[0:8], header)
	binary.BigEndian.PutUint16(data[8:10], r.Type)
	binary.BigEndian.PutUint16(data[10:12], r.Code)
	copy(data[12:], r.Data)

	return data, nil
}

func (r *ErrorMessage) UnmarshalBinary(data []byte) error {
	header := &Header{}
	if err := header.UnmarshalBinary(data); err != nil {
		return err
	}
	if len(data) != int(header.Length) {
		return ErrInvalidPacketLength
	}

	r.Header = *header
	r.Type = binary.BigEndian.Uint16(data[8:10])
	r.Code = binary.BigEndian.Uint16(data[10:12])
	r.Data = data[12:]

	return nil
}
