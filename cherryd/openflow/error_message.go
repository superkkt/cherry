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
	Type ErrorType
	Code ErrorCode
	Data []byte
}

func (r *ErrorMessage) MarshalBinary() ([]byte, error) {
	var length uint16 = 12 // header length + type + code
	if r.Data != nil {
		length += uint16(len(r.Data))
	}

	r.Header.Length = length
	header, err := r.Header.MarshalBinary()
	if err != nil {
		return nil, err
	}

	data := make([]byte, length)
	copy(data[0:8], header)
	binary.BigEndian.PutUint16(data[8:10], uint16(r.Type))
	binary.BigEndian.PutUint16(data[10:12], uint16(r.Code))
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
	r.Type = ErrorType(binary.BigEndian.Uint16(data[8:10]))
	r.Code = ErrorCode(binary.BigEndian.Uint16(data[10:12]))
	r.Data = data[12:]

	return nil
}
