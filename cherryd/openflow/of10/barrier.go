/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of10

import (
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type BarrierRequest struct {
	openflow.Message
}

func NewBarrierRequest(xid uint32) openflow.BarrierRequest {
	return &BarrierRequest{
		Message: openflow.NewMessage(openflow.OF10_VERSION, OFPT_BARRIER_REQUEST, xid),
	}
}

func (r *BarrierRequest) MarshalBinary() ([]byte, error) {
	return r.Message.MarshalBinary()
}

type BarrierReply struct {
	openflow.Message
}

func (r *BarrierReply) UnmarshalBinary(data []byte) error {
	return r.Message.UnmarshalBinary(data)
}
