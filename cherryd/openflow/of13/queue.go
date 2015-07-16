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

package of13

import (
	"encoding/binary"
	"github.com/superkkt/cherry/cherryd/openflow"
)

type QueueGetConfigRequest struct {
	openflow.Message
	port openflow.OutPort
}

func NewQueueGetConfigRequest(xid uint32) openflow.QueueGetConfigRequest {
	return &QueueGetConfigRequest{
		Message: openflow.NewMessage(openflow.OF13_VERSION, OFPT_QUEUE_GET_CONFIG_REQUEST, xid),
	}
}

func (r *QueueGetConfigRequest) Port() openflow.OutPort {
	return r.port
}

func (r *QueueGetConfigRequest) SetPort(p openflow.OutPort) {
	r.port = p
}

func (r *QueueGetConfigRequest) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	if r.port.IsNone() {
		binary.BigEndian.PutUint32(v[0:4], OFPP_ANY)
	} else {
		binary.BigEndian.PutUint32(v[0:4], r.port.Value())
	}
	// v[4:8] is padding
	r.SetPayload(v)

	return r.Message.MarshalBinary()
}
