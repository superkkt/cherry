/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of10

import (
	"encoding/binary"
	"errors"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type FlowStatsRequest struct {
	openflow.Message
	match   openflow.Match
	tableID uint8
}

func NewFlowStatsRequest(xid uint32) openflow.FlowStatsRequest {
	return &FlowStatsRequest{
		Message: openflow.NewMessage(openflow.OF10_VERSION, OFPT_STATS_REQUEST, xid),
	}
}

func (r FlowStatsRequest) Cookie() uint64 {
	// OpenFlow 1.0 does not have cookie
	return 0
}

func (r *FlowStatsRequest) SetCookie(cookie uint64) error {
	// OpenFlow 1.0 does not have cookie
	return nil
}

func (r FlowStatsRequest) CookieMask() uint64 {
	// OpenFlow 1.0 does not have cookie
	return 0
}

func (r *FlowStatsRequest) SetCookieMask(mask uint64) error {
	// OpenFlow 1.0 does not have cookie
	return nil
}

func (r FlowStatsRequest) Match() openflow.Match {
	return r.match
}

func (r *FlowStatsRequest) SetMatch(match openflow.Match) error {
	if match == nil {
		return errors.New("match is nil")
	}
	r.match = match
	return nil
}

func (r FlowStatsRequest) TableID() uint8 {
	return r.tableID
}

// 0xFF means all table
func (r *FlowStatsRequest) SetTableID(id uint8) error {
	r.tableID = id
	return nil
}

// TODO: Need testing
func (r *FlowStatsRequest) MarshalBinary() ([]byte, error) {
	v := make([]byte, 48)
	binary.BigEndian.PutUint16(v[0:2], OFPST_FLOW)
	// v[2:4] is flags, but not yet defined
	match, err := r.match.MarshalBinary()
	if err != nil {
		return nil, err
	}
	copy(v[4:44], match)
	v[44] = r.tableID
	// v[45] is padding
	binary.BigEndian.PutUint16(v[46:48], OFPP_NONE)
	r.SetPayload(v)

	return r.Message.MarshalBinary()
}

// TODO: Implement FlowStatsReply
