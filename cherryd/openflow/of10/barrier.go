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
	header openflow.Header
}

func NewBarrierRequest(xid uint32) *BarrierRequest {
	return &BarrierRequest{
		header: openflow.Header{
			Version: openflow.Ver10,
			Type:    OFPT_BARRIER_REQUEST,
			XID:     xid,
		},
	}
}

func (r *BarrierRequest) Header() openflow.Header {
	return r.header
}

func (r *BarrierRequest) MarshalBinary() ([]byte, error) {
	r.header.Length = 8
	return r.header.MarshalBinary()
}

func (r *BarrierRequest) UnmarshalBinary(data []byte) error {
	return openflow.ErrUnsupportedUnmarshaling
}

type BarrierReply struct {
	header openflow.Header
}

func (r *BarrierReply) Header() openflow.Header {
	return r.header
}

func (r *BarrierReply) MarshalBinary() ([]byte, error) {
	return nil, openflow.ErrUnsupportedMarshaling
}

func (r *BarrierReply) UnmarshalBinary(data []byte) error {
	return r.header.UnmarshalBinary(data)
}
