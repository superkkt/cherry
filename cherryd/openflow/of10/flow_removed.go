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
	match           openflow.Match
	cookie          uint64
	priority        uint16
	reason          uint8
	durationSec     uint32
	durationNanoSec uint32
	idleTimeout     uint16
	packetCount     uint64
	byteCount       uint64
}

func (r FlowRemoved) Cookie() uint64 {
	return r.cookie
}

func (r FlowRemoved) Priority() uint16 {
	return r.priority
}

func (r FlowRemoved) Reason() uint8 {
	return r.reason
}

func (r FlowRemoved) TableID() uint8 {
	// OpenFlow 1.0 does not have table ID
	return 0
}

func (r FlowRemoved) DurationSec() uint32 {
	return r.durationSec
}

func (r FlowRemoved) DurationNanoSec() uint32 {
	return r.durationNanoSec
}

func (r FlowRemoved) IdleTimeout() uint16 {
	return r.idleTimeout
}

func (r FlowRemoved) HardTimeout() uint16 {
	// OpenFlow 1.0 does not have hard timeout value in the flow removed message
	return 0
}

func (r FlowRemoved) PacketCount() uint64 {
	return r.packetCount
}

func (r FlowRemoved) ByteCount() uint64 {
	return r.byteCount
}

func (r FlowRemoved) Match() openflow.Match {
	return r.match
}

func (r *FlowRemoved) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 80 {
		return openflow.ErrInvalidPacketLength
	}
	r.match = NewMatch()
	if err := r.match.UnmarshalBinary(payload[0:40]); err != nil {
		return err
	}
	r.cookie = binary.BigEndian.Uint64(payload[40:48])
	r.priority = binary.BigEndian.Uint16(payload[48:50])
	r.reason = payload[50]
	// payload[51] is padding
	r.durationSec = binary.BigEndian.Uint32(payload[52:56])
	r.durationNanoSec = binary.BigEndian.Uint32(payload[56:60])
	r.idleTimeout = binary.BigEndian.Uint16(payload[60:62])
	// payload[62:64] is padding
	r.packetCount = binary.BigEndian.Uint64(payload[64:72])
	r.byteCount = binary.BigEndian.Uint64(payload[72:80])

	return nil
}
