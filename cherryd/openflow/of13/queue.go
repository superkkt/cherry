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

// QueueGetConfigReply implementations by ksang:
type QueueProperty struct {
	typ          openflow.PropertyType
	length       uint16
	rate         uint16
	experimenter uint32
	data         []byte
}

func NewQueueProperty() openflow.QueueProperty {
	return &QueueProperty{}
}

func (r *QueueProperty) Type() openflow.PropertyType {
	return r.typ
}

func (r *QueueProperty) Length() uint16 {
	return r.length
}

func (r *QueueProperty) Rate() (uint16, error) {
	if r.typ != openflow.OFPQT_MIN_RATE || r.typ != openflow.OFPQT_MAX_RATE {
		return 0x0, openflow.ErrInvalidPropertyMethod
	}
	return r.rate, nil
}

func (r *QueueProperty) Experimenter() (uint32, error) {
	if r.typ != openflow.OFPQT_EXPERIMENTER {
		return 0x0, openflow.ErrInvalidPropertyMethod
	}
	return r.experimenter, nil
}

func (r *QueueProperty) Data() []byte {
	return r.data
}

func (r *QueueProperty) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return openflow.ErrInvalidPacketLength
	}
	r.typ = openflow.PropertyType(binary.BigEndian.Uint16(data[0:2]))
	r.length = binary.BigEndian.Uint16(data[2:4])
	// data[4:8] is pad
	if len(data) < int(r.length) || int(r.length) < 16 {
		return openflow.ErrInvalidPacketLength
	}
	switch r.typ {
	case openflow.OFPQT_MIN_RATE:
		r.rate = binary.BigEndian.Uint16(data[8:10])
	case openflow.OFPQT_MAX_RATE:
		r.rate = binary.BigEndian.Uint16(data[8:10])
	case openflow.OFPQT_EXPERIMENTER:
		r.experimenter = binary.BigEndian.Uint32(data[8:12])
		// data[12:16] is pad
		r.data = make([]byte, int(r.length)-16)
		copy(r.data, data[16:int(r.length)])
	}
	return nil
}

type Queue struct {
	id       uint32
	port     uint32
	length   uint16
	property []openflow.QueueProperty
}

func (r *Queue) ID() uint32 {
	return r.id
}

func (r *Queue) Port() uint32 {
	return r.port
}

func (r *Queue) Length() uint16 {
	return r.length
}

func (r *Queue) Property() []openflow.QueueProperty {
	return r.property
}

func (r *Queue) UnmarshalBinary(data []byte) error {
	if len(data) < 16 {
		return openflow.ErrInvalidPacketLength
	}
	r.id = binary.BigEndian.Uint32(data[0:4])
	r.port = binary.BigEndian.Uint32(data[4:8])
	r.length = binary.BigEndian.Uint16(data[8:10])
	// data[10:16] is pad
	if len(data) < int(r.length) || int(r.length) < 16 {
		return openflow.ErrInvalidPacketLength
	}
	for i := 16; i < int(r.length); {
		p := NewQueueProperty()
		if err := p.UnmarshalBinary(data[i:]); err != nil {
			return err
		}
		r.property = append(r.property, p)
		i += int(p.Length())
	}
	return nil
}

func NewQueue() openflow.Queue {
	return &Queue{}
}

type QueueGetConfigReply struct {
	openflow.Message
	port  uint32
	queue []openflow.Queue
}

func (r *QueueGetConfigReply) Port() uint32 {
	return r.port
}

func (r *QueueGetConfigReply) Queue() []openflow.Queue {
	return r.queue
}

func (r *QueueGetConfigReply) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}
	payload := r.Payload()
	if payload == nil || len(payload) < 8 {
		return openflow.ErrInvalidPacketLength
	}
	r.port = binary.BigEndian.Uint32(payload[0:4])
	// Unmarshal Queues
	for i := 8; i < len(payload); {
		q := NewQueue()
		if err := q.UnmarshalBinary(payload[i:]); err != nil {
			return err
		}
		r.queue = append(r.queue, q)
		i += int(q.Length())
	}
	return nil
}
