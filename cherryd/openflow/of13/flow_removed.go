/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

import (
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
	// TODO: Implement this
	// Match *Match
}

func (r *FlowRemoved) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	// TODO: Implement this function
	//payload := r.Payload()
	//	if payload == nil || len(payload) < 56 {
	//		return openflow.ErrInvalidPacketLength
	//	}

	return nil
}
