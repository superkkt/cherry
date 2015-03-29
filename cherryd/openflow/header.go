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

type Header struct {
	Version uint8
	Type    PacketType
	Length  uint16
	Xid     uint32
}

func (r *Header) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	v[0] = r.Version
	v[1] = uint8(r.Type)
	binary.BigEndian.PutUint16(v[2:4], r.Length)
	binary.BigEndian.PutUint32(v[4:8], r.Xid)

	return v, nil
}

func (r *Header) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return ErrInvalidPacketLength
	}

	r.Version = data[0]
	r.Type = PacketType(data[1])
	r.Length = binary.BigEndian.Uint16(data[2:4])
	r.Xid = binary.BigEndian.Uint32(data[4:8])

	return nil
}
