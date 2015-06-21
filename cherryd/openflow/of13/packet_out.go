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

type PacketOut struct {
	err error
	openflow.Message
	inPort openflow.InPort
	action openflow.Action
	data   []byte
}

func NewPacketOut(xid uint32) openflow.PacketOut {
	return &PacketOut{
		Message: openflow.NewMessage(openflow.OF13_VERSION, OFPT_PACKET_OUT, xid),
	}
}

func (r *PacketOut) Error() error {
	return r.err
}

func (r *PacketOut) InPort() openflow.InPort {
	return r.inPort
}

func (r *PacketOut) SetInPort(port openflow.InPort) {
	r.inPort = port
}

func (r *PacketOut) Action() openflow.Action {
	return r.action
}

func (r *PacketOut) SetAction(action openflow.Action) {
	if action == nil {
		panic("action is nil")
	}
	r.action = action
}

func (r *PacketOut) Data() []byte {
	return r.data
}

func (r *PacketOut) SetData(data []byte) {
	if data == nil {
		panic("data is nil")
	}
	r.data = data
}

func (r *PacketOut) MarshalBinary() ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}

	action := make([]byte, 0)
	if r.action != nil {
		a, err := r.action.MarshalBinary()
		if err != nil {
			return nil, err
		}
		action = append(action, a...)
	}

	v := make([]byte, 16)
	binary.BigEndian.PutUint32(v[0:4], OFP_NO_BUFFER)
	port := r.inPort.Value()
	if r.inPort.IsController() {
		port = OFPP_CONTROLLER
	}
	binary.BigEndian.PutUint32(v[4:8], port)
	binary.BigEndian.PutUint16(v[8:10], uint16(len(action)))
	// v[10:16] is padding
	v = append(v, action...)
	if r.data != nil && len(r.data) > 0 {
		v = append(v, r.data...)
	}

	r.SetPayload(v)
	return r.Message.MarshalBinary()
}
