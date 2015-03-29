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

type PacketInMessage struct {
	Header
	BufferID uint32
	Length   uint16
	InPort   uint16
	Reason   PacketInReason
	Data     []byte
}

func (r *PacketInMessage) UnmarshalBinary(data []byte) error {
	header := &Header{}
	if err := header.UnmarshalBinary(data); err != nil {
		return err
	}
	if len(data) < 18 || len(data) != int(header.Length) {
		return ErrInvalidPacketLength
	}

	r.Header = *header
	r.BufferID = binary.BigEndian.Uint32(data[8:12])
	r.Length = binary.BigEndian.Uint16(data[12:14])
	r.InPort = binary.BigEndian.Uint16(data[14:16])
	r.Reason = PacketInReason(data[16])
	r.Data = data[18:]

	return nil
}
