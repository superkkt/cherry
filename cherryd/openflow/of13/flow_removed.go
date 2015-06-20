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

type FlowRemoved struct {
	openflow.Message
	cookie          uint64
	priority        uint16
	reason          uint8
	tableID         uint8
	durationSec     uint32
	durationNanoSec uint32
	idleTimeout     uint16
	hardTimeout     uint16
	packetCount     uint64
	byteCount       uint64
	match           openflow.Match
}

func (r FlowRemoved) Cookie() uint64 {
	return r.cookie
}

func (r FlowRemoved) Priority() uint16 {
	return r.priority
}

func (r FlowRemoved) Reason() uint8 {
	return r.reason
}

func (r FlowRemoved) TableID() uint8 {
	return r.tableID
}

func (r FlowRemoved) DurationSec() uint32 {
	return r.durationSec
}

func (r FlowRemoved) DurationNanoSec() uint32 {
	return r.durationNanoSec
}

func (r FlowRemoved) IdleTimeout() uint16 {
	return r.idleTimeout
}

func (r FlowRemoved) HardTimeout() uint16 {
	return r.hardTimeout
}

func (r FlowRemoved) PacketCount() uint64 {
	return r.packetCount
}

func (r FlowRemoved) ByteCount() uint64 {
	return r.byteCount
}

func (r FlowRemoved) Match() openflow.Match {
	return r.match
}

func (r *FlowRemoved) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 48 {
		return openflow.ErrInvalidPacketLength
	}
	r.cookie = binary.BigEndian.Uint64(payload[0:8])
	r.priority = binary.BigEndian.Uint16(payload[8:10])
	r.reason = payload[10]
	r.tableID = payload[11]
	r.durationSec = binary.BigEndian.Uint32(payload[12:16])
	r.durationNanoSec = binary.BigEndian.Uint32(payload[16:20])
	r.idleTimeout = binary.BigEndian.Uint16(payload[20:22])
	r.hardTimeout = binary.BigEndian.Uint16(payload[22:24])
	r.packetCount = binary.BigEndian.Uint64(payload[24:32])
	r.byteCount = binary.BigEndian.Uint64(payload[32:40])
	r.match = NewMatch()
	if err := r.match.UnmarshalBinary(payload[40:]); err != nil {
		return err
	}

	return nil
}
