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
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type PortDescRequest struct {
	openflow.Message
}

func NewPortDescRequest(xid uint32) openflow.PortDescRequest {
	return &PortDescRequest{
		Message: openflow.NewMessage(openflow.OF13_VERSION, OFPT_MULTIPART_REQUEST, xid),
	}
}

func (r *PortDescRequest) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	// Multipart description request
	binary.BigEndian.PutUint16(v[0:2], OFPMP_PORT_DESC)
	// No flags and body
	r.SetPayload(v)

	return r.Message.MarshalBinary()
}

type PortDescReply struct {
	openflow.Message
	ports []openflow.Port
}

func (r PortDescReply) Ports() []openflow.Port {
	return r.ports
}

func (r *PortDescReply) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 8 {
		return openflow.ErrInvalidPacketLength
	}
	nPorts := (len(payload) - 8) / 64
	if nPorts == 0 {
		return nil
	}
	r.ports = make([]openflow.Port, nPorts)
	for i := 0; i < nPorts; i++ {
		buf := payload[8+i*64:]
		r.ports[i] = new(Port)
		if err := r.ports[i].UnmarshalBinary(buf[0:64]); err != nil {
			return err
		}
	}

	return nil
}
