/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of10

import (
	"encoding/binary"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type FlowRemoved struct {
	openflow.Message
	Match           openflow.Match
	Cookie          uint64
	Priority        uint16
	Reason          uint8
	DurationSec     uint32
	DurationNanoSec uint32
	IdleTimeout     uint16
	PacketCount     uint64
	ByteCount       uint64
}

func (r *FlowRemoved) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 80 {
		return openflow.ErrInvalidPacketLength
	}
	r.Match = NewMatch()
	if err := r.Match.UnmarshalBinary(payload[0:40]); err != nil {
		return err
	}
	r.Cookie = binary.BigEndian.Uint64(payload[40:48])
	r.Priority = binary.BigEndian.Uint16(payload[48:50])
	r.Reason = payload[50]
	// payload[51] is padding
	r.DurationSec = binary.BigEndian.Uint32(payload[52:56])
	r.DurationNanoSec = binary.BigEndian.Uint32(payload[56:60])
	r.IdleTimeout = binary.BigEndian.Uint16(payload[60:62])
	// payload[62:64] is padding
	r.PacketCount = binary.BigEndian.Uint64(payload[64:72])
	r.ByteCount = binary.BigEndian.Uint64(payload[72:80])

	return nil
}
