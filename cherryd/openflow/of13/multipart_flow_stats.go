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

type FlowStatsRequest struct {
	openflow.Message
	TableID            uint8
	Cookie, CookieMask uint64
	Match              openflow.Match
}

func NewFlowStatsRequest(xid uint32, tableID uint8, cookie, mask uint64, match openflow.Match) *FlowStatsRequest {
	if match == nil {
		panic("NIL match parameter!")
	}

	return &FlowStatsRequest{
		Message:    openflow.NewMessage(openflow.Ver13, OFPT_MULTIPART_REQUEST, xid),
		TableID:    tableID,
		Cookie:     cookie,
		CookieMask: mask,
		Match:      match,
	}
}

func (r *FlowStatsRequest) MarshalBinary() ([]byte, error) {
	v := make([]byte, 40)
	// Flow stats request
	binary.BigEndian.PutUint16(v[0:2], OFPMP_FLOW)
	v[8] = r.TableID
	// v[9:12] is padding
	binary.BigEndian.PutUint32(v[12:16], OFPP_ANY)
	binary.BigEndian.PutUint32(v[16:20], OFPG_ANY)
	// v[20:24] is padding
	binary.BigEndian.PutUint64(v[24:32], r.Cookie)
	binary.BigEndian.PutUint64(v[32:40], r.CookieMask)
	match, err := r.Match.MarshalBinary()
	if err != nil {
		return nil, err
	}
	v = append(v, match...)
	r.SetPayload(v)

	return r.Message.MarshalBinary()
}

// TODO: Implement TableFeaturesReply
