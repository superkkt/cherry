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

type FlowRemoved struct {
	openflow.Message
	Cookie          uint64
	Priority        uint16
	Reason          uint8
	TableID         uint8
	DurationSec     uint32
	DurationNanoSec uint32
	IdleTimeout     uint16
	HardTimeout     uint16
	PacketCount     uint64
	ByteCount       uint64
	Match           openflow.Match
}

func (r *FlowRemoved) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 48 {
		return openflow.ErrInvalidPacketLength
	}
	r.Cookie = binary.BigEndian.Uint64(payload[0:8])
	r.Priority = binary.BigEndian.Uint16(payload[8:10])
	r.Reason = payload[10]
	r.TableID = payload[11]
	r.DurationSec = binary.BigEndian.Uint32(payload[12:16])
	r.DurationNanoSec = binary.BigEndian.Uint32(payload[16:20])
	r.IdleTimeout = binary.BigEndian.Uint16(payload[20:22])
	r.HardTimeout = binary.BigEndian.Uint16(payload[22:24])
	r.PacketCount = binary.BigEndian.Uint64(payload[24:32])
	r.ByteCount = binary.BigEndian.Uint64(payload[32:40])
	r.Match = NewMatch()
	if err := r.Match.UnmarshalBinary(payload[40:]); err != nil {
		return err
	}

	return nil
}
