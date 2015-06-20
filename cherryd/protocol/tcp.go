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

package protocol

import (
	"encoding/binary"
	"errors"
	"net"
)

type TCP struct {
	srcIP          net.IP
	dstIP          net.IP
	SrcPort        uint16
	DstPort        uint16
	Sequence       uint32
	Acknowledgment uint32
	// From LSB, FIN, SYN, RST, PSH, ACK, URG, ECE, CWR, and NS.
	Flags      uint16
	WindowSize uint16
	Checksum   uint16
	Urgent     uint16
	Payload    []byte
}

// TCP checksum needs a pseudo header that has src and dst IPv4 addresses.
func (r *TCP) SetPseudoHeader(src, dst net.IP) {
	r.srcIP = src
	r.dstIP = dst
}

func (r TCP) MarshalBinary() ([]byte, error) {
	length := 20
	if r.Payload != nil {
		length += len(r.Payload)
	}

	v := make([]byte, length)
	binary.BigEndian.PutUint16(v[0:2], r.SrcPort)
	binary.BigEndian.PutUint16(v[2:4], r.DstPort)
	binary.BigEndian.PutUint32(v[4:8], r.Sequence)
	binary.BigEndian.PutUint32(v[8:12], r.Acknowledgment)
	v[12] = uint8(0x5<<4 | (r.Flags >> 8 & 0x1))
	v[13] = uint8(r.Flags & 0xFF)
	binary.BigEndian.PutUint16(v[14:16], r.WindowSize)
	// v[16:18] is checksum
	binary.BigEndian.PutUint16(v[18:20], r.Urgent)
	if r.Payload != nil {
		copy(v[20:], r.Payload)
	}

	if r.srcIP == nil || r.dstIP == nil {
		return nil, errors.New("nil pseudo IP addresses")
	}
	pseudo := make([]byte, 12)
	copy(pseudo[0:4], r.srcIP)
	copy(pseudo[4:8], r.dstIP)
	pseudo[9] = 6 // TCP
	binary.BigEndian.PutUint16(pseudo[10:12], uint16(length))

	checksum := calculateChecksum(append(pseudo, v...))
	binary.BigEndian.PutUint16(v[16:18], checksum)

	return v, nil
}

func (r *TCP) UnmarshalBinary(data []byte) error {
	if len(data) < 20 {
		return errors.New("invalid TCP packet length")
	}

	r.SrcPort = binary.BigEndian.Uint16(data[0:2])
	r.DstPort = binary.BigEndian.Uint16(data[2:4])
	r.Sequence = binary.BigEndian.Uint32(data[4:8])
	r.Acknowledgment = binary.BigEndian.Uint32(data[8:12])
	offset := int((data[12] >> 4)) * 4
	r.Flags = uint16((data[12]&0x1)<<8 | data[13])
	r.WindowSize = binary.BigEndian.Uint16(data[14:16])
	r.Checksum = binary.BigEndian.Uint16(data[16:18])
	r.Urgent = binary.BigEndian.Uint16(data[18:20])
	if len(data) > offset {
		r.Payload = data[offset:]
	}

	return nil
}
