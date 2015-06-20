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
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type BarrierRequest struct {
	openflow.Message
}

func NewBarrierRequest(xid uint32) openflow.BarrierRequest {
	return &BarrierRequest{
		Message: openflow.NewMessage(openflow.OF13_VERSION, OFPT_BARRIER_REQUEST, xid),
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
