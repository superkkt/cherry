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
	header          openflow.Header
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

func (r *FlowRemoved) Header() openflow.Header {
	return r.header
}

func (r *FlowRemoved) MarshalBinary() ([]byte, error) {
	return nil, openflow.ErrUnsupportedMarshaling
}

func (r *FlowRemoved) UnmarshalBinary(data []byte) error {
	// TODO: Implement this function
	return nil
}
