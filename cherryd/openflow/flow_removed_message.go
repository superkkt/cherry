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

type FlowRemovedMessage struct {
	Header
	Match           *FlowMatch
	Cookie          uint64
	Priority        uint16
	Reason          FlowRemovedReason
	DurationSec     uint32
	DurationNanoSec uint32
	IdleTimeout     uint16
	PacketCount     uint64
	ByteCount       uint64
}

func (r *FlowRemovedMessage) UnmarshalBinary(data []byte) error {
	header := &Header{}
	if err := header.UnmarshalBinary(data); err != nil {
		return err
	}
	if len(data) < 88 || len(data) != int(header.Length) {
		return ErrInvalidPacketLength
	}

	r.Header = *header
	r.Match = &FlowMatch{}
	if err := r.Match.UnmarshalBinary(data[8:48]); err != nil {
		return err
	}
	r.Cookie = binary.BigEndian.Uint64(data[48:56])
	r.Priority = binary.BigEndian.Uint16(data[56:58])
	r.Reason = FlowRemovedReason(data[58])
	// data[59] is padding
	r.DurationSec = binary.BigEndian.Uint32(data[60:64])
	r.DurationNanoSec = binary.BigEndian.Uint32(data[64:68])
	r.IdleTimeout = binary.BigEndian.Uint16(data[68:70])
	r.PacketCount = binary.BigEndian.Uint64(data[72:80])
	r.ByteCount = binary.BigEndian.Uint64(data[80:88])

	return nil
}
