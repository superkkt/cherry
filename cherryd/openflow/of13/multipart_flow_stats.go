/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
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

package of13

import (
	"encoding/binary"
	"errors"
	"github.com/superkkt/cherry/cherryd/openflow"
)

type FlowStatsRequest struct {
	err error
	openflow.Message
	tableID            uint8
	cookie, cookieMask uint64
	match              openflow.Match
}

func NewFlowStatsRequest(xid uint32) openflow.FlowStatsRequest {
	return &FlowStatsRequest{
		Message: openflow.NewMessage(openflow.OF13_VERSION, OFPT_MULTIPART_REQUEST, xid),
	}
}

func (r *FlowStatsRequest) Error() error {
	return r.err
}

func (r *FlowStatsRequest) Cookie() uint64 {
	return r.cookie
}

func (r *FlowStatsRequest) SetCookie(cookie uint64) {
	r.cookie = cookie
}

func (r *FlowStatsRequest) CookieMask() uint64 {
	return r.cookieMask
}

func (r *FlowStatsRequest) SetCookieMask(mask uint64) {
	r.cookieMask = mask
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

func (r *FlowStatsRequest) MarshalBinary() ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}

	v := make([]byte, 40)
	// Flow stats request
	binary.BigEndian.PutUint16(v[0:2], OFPMP_FLOW)
	v[8] = r.tableID
	// v[9:12] is padding
	binary.BigEndian.PutUint32(v[12:16], OFPP_ANY)
	binary.BigEndian.PutUint32(v[16:20], OFPG_ANY)
	// v[20:24] is padding
	binary.BigEndian.PutUint64(v[24:32], r.cookie)
	binary.BigEndian.PutUint64(v[32:40], r.cookieMask)

	if r.match == nil {
		return nil, errors.New("empty flow match")
	}
	match, err := r.match.MarshalBinary()
	if err != nil {
		return nil, err
	}
	v = append(v, match...)
	r.SetPayload(v)

	return r.Message.MarshalBinary()
}

// TODO: Implement FlowStatsReply
