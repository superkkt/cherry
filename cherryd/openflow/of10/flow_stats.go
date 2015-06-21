/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service, Inc. All rights reserved. 
 * Kitae Kim <superkkt@sds.co.kr>
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along
 * with this program; if not, write to the Free Software Foundation, Inc.,
 * 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
 */

package of10

import (
	"encoding/binary"
	"errors"
	"github.com/superkkt/cherry/cherryd/openflow"
)

type FlowStatsRequest struct {
	err error
	openflow.Message
	match   openflow.Match
	tableID uint8
}

func NewFlowStatsRequest(xid uint32) openflow.FlowStatsRequest {
	return &FlowStatsRequest{
		Message: openflow.NewMessage(openflow.OF10_VERSION, OFPT_STATS_REQUEST, xid),
	}
}

func (r *FlowStatsRequest) Error() error {
	return r.err
}

func (r *FlowStatsRequest) Cookie() uint64 {
	// OpenFlow 1.0 does not have cookie
	return 0
}

func (r *FlowStatsRequest) SetCookie(cookie uint64) {
	// OpenFlow 1.0 does not have cookie
}

func (r *FlowStatsRequest) CookieMask() uint64 {
	// OpenFlow 1.0 does not have cookie
	return 0
}

func (r *FlowStatsRequest) SetCookieMask(mask uint64) {
	// OpenFlow 1.0 does not have cookie
}

func (r *FlowStatsRequest) Match() openflow.Match {
	return r.match
}

func (r *FlowStatsRequest) SetMatch(match openflow.Match) {
	if match == nil {
		panic("match is nil")
	}
	r.match = match
}

func (r *FlowStatsRequest) TableID() uint8 {
	return r.tableID
}

// 0xFF means all table
func (r *FlowStatsRequest) SetTableID(id uint8) {
	r.tableID = id
}

// TODO: Need testing
func (r *FlowStatsRequest) MarshalBinary() ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}

	v := make([]byte, 48)
	binary.BigEndian.PutUint16(v[0:2], OFPST_FLOW)
	// v[2:4] is flags, but not yet defined

	if r.match == nil {
		return nil, errors.New("empty flow match")
	}
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
