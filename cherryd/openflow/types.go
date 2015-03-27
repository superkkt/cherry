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
)

type Header struct {
	Version uint8
	Type    uint8
	Length  uint16
	Xid     uint32
}

func (r *Header) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	v[0] = r.Version
	v[1] = r.Type
	binary.BigEndian.PutUint16(v[2:4], r.Length)
	binary.BigEndian.PutUint32(v[4:8], r.Xid)

	return v, nil
}

func (r *Header) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return ErrInvalidPacketLength
	}

	r.Version = data[0]
	r.Type = data[1]
	r.Length = binary.BigEndian.Uint16(data[2:4])
	r.Xid = binary.BigEndian.Uint32(data[4:8])

	return nil
}

type HelloMessage struct {
	Header
	// This message does not contain a body beyond the header
}

func (r *HelloMessage) MarshalBinary() ([]byte, error) {
	return r.Header.MarshalBinary()
}

func (r *HelloMessage) UnmarshalBinary(data []byte) error {
	return r.Header.UnmarshalBinary(data)
}

type FeaturesRequestMessage struct {
	Header
	// This message does not contain a body beyond the header
}

func (r *FeaturesRequestMessage) MarshalBinary() ([]byte, error) {
	return r.Header.MarshalBinary()
}

func (r *FeaturesRequestMessage) UnmarshalBinary(data []byte) error {
	return r.Header.UnmarshalBinary(data)
}

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
