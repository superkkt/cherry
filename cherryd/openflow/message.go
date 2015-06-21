/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service 
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

package openflow

import (
	"encoding/binary"
)

type Header interface {
	Version() uint8
	Type() uint8
	TransactionID() uint32
	SetTransactionID(xid uint32)
}

type Message struct {
	version uint8
	msgType uint8
	xid     uint32
	length  uint16
	payload []byte
}

func NewMessage(version uint8, msgType uint8, xid uint32) Message {
	return Message{
		version: version,
		msgType: msgType,
		xid:     xid,
		length:  8,
	}
}

func (r *Message) Version() uint8 {
	return r.version
}

func (r *Message) Type() uint8 {
	return r.msgType
}

func (r *Message) TransactionID() uint32 {
	return r.xid
}

func (r *Message) SetTransactionID(xid uint32) {
	r.xid = xid
}

func (r *Message) Payload() []byte {
	return r.payload
}

func (r *Message) SetPayload(payload []byte) {
	r.payload = payload
	if payload == nil {
		r.length = 8
	} else {
		r.length = uint16(8 + len(payload))
	}
}

func (r *Message) MarshalBinary() ([]byte, error) {
	var length uint16 = 8
	if r.payload != nil {
		length += uint16(len(r.payload))
	}

	v := make([]byte, length)
	v[0] = r.version
	v[1] = r.msgType
	binary.BigEndian.PutUint16(v[2:4], length)
	binary.BigEndian.PutUint32(v[4:8], r.xid)
	if length > 8 {
		copy(v[8:], r.payload)
	}

	return v, nil
}

func (r *Message) UnmarshalBinary(data []byte) error {
	if data == nil || len(data) < 8 {
		return ErrInvalidPacketLength
	}

	r.version = data[0]
	r.msgType = data[1]
	r.length = binary.BigEndian.Uint16(data[2:4])
	if r.length < 8 || len(data) < int(r.length) {
		return ErrInvalidPacketLength
	}
	r.xid = binary.BigEndian.Uint32(data[4:8])
	r.payload = data[8:r.length]

	return nil
}
