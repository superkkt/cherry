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
	"github.com/superkkt/cherry/cherryd/openflow"
)

type PacketIn struct {
	openflow.Message
	bufferID uint32
	length   uint16
	inPort   uint16
	reason   uint8
	data     []byte
}

func (r PacketIn) BufferID() uint32 {
	return r.bufferID
}

func (r PacketIn) InPort() uint32 {
	return uint32(r.inPort)
}

func (r PacketIn) Data() []byte {
	return r.data
}

func (r PacketIn) Length() uint16 {
	return r.length
}

func (r PacketIn) TableID() uint8 {
	// OpenFlow 1.0 does not have table ID
	return 0
}

func (r PacketIn) Reason() uint8 {
	return r.reason
}

func (r PacketIn) Cookie() uint64 {
	// OpenFlow 1.0 does not have cookie
	return 0
}

func (r *PacketIn) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 10 {
		return openflow.ErrInvalidPacketLength
	}
	r.bufferID = binary.BigEndian.Uint32(payload[0:4])
	r.length = binary.BigEndian.Uint16(payload[4:6])
	r.inPort = binary.BigEndian.Uint16(payload[6:8])
	r.reason = payload[8]
	// payload[9] is padding
	if len(payload) >= 10 {
		// TODO: Check data size by comparing with r.Length
		r.data = payload[10:]
	}

	return nil
}
