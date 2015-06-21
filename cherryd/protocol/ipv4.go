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

package protocol

import (
	"encoding/binary"
	"errors"
	"net"
)

// TODO: Test this module!!

type IPv4 struct {
	Version  uint8
	IHL      uint8
	DSCP     uint8
	ECN      uint8
	Length   uint16
	ID       uint16
	Flags    uint8
	Offset   uint16
	TTL      uint8
	Protocol uint8
	Checksum uint16
	SrcIP    net.IP
	DstIP    net.IP
	Payload  []byte
}

func NewIPv4(src, dst net.IP, protocol uint8, payload []byte) *IPv4 {
	switch protocol {
	case 1: // ICMP
	case 2: // IGMP
	case 6: // TCP
	case 17: // UDP
	case 41: // IPv6 encapsulation
	case 89: // OSPF
	case 132: // SCTP
	default:
		panic("unknown protocol number")
	}

	if len(payload) > 0xFFFF-20 {
		panic("payload is too long")
	}

	return &IPv4{
		Version: 4,
		IHL:     5,                         // 20 bytes
		Length:  uint16(len(payload) + 20), // Payload + Header
		// FIXME: Should we set ID as a random number?
		Flags:    0x2, // Don't Fragment
		TTL:      64,
		Protocol: protocol,
		SrcIP:    src,
		DstIP:    dst,
		Payload:  payload,
	}
}

func (r IPv4) MarshalBinary() ([]byte, error) {
	if r.SrcIP == nil || r.DstIP == nil {
		return nil, errors.New("nil IP address")
	}

	header := make([]byte, 20)
	header[0] = (r.Version&0xF)<<4 | r.IHL&0xF
	header[1] = (r.DSCP&0x3F)<<2 | r.ECN&0x3
	binary.BigEndian.PutUint16(header[2:4], r.Length)
	binary.BigEndian.PutUint16(header[4:6], r.ID)
	binary.BigEndian.PutUint16(header[6:8], (uint16(r.Flags)&0x7)<<13|r.Offset&0x1FFF)
	header[8] = r.TTL
	header[9] = r.Protocol
	// header[10:12] = checksum
	copy(header[12:16], r.SrcIP)
	copy(header[16:20], r.DstIP)

	checksum := calculateChecksum(header)
	binary.BigEndian.PutUint16(header[10:12], checksum)

	if r.Payload == nil {
		return header, nil
	}
	return append(header, r.Payload...), nil
}

func (r *IPv4) UnmarshalBinary(data []byte) error {
	if len(data) < 20 {
		return errors.New("invalid IPv4 packet length")
	}

	r.Version = (data[0] >> 4) & 0xF
	r.IHL = data[0] & 0xF
	r.DSCP = (data[1] >> 2) & 0x3F
	r.ECN = data[1] & 0x3
	r.Length = binary.BigEndian.Uint16(data[2:4])
	r.ID = binary.BigEndian.Uint16(data[4:6])
	v := binary.BigEndian.Uint16(data[6:8])
	r.Flags = uint8((v >> 13) & 0x7)
	r.Offset = v & 0x1FFF
	r.TTL = data[8]
	r.Protocol = data[9]
	r.Checksum = binary.BigEndian.Uint16(data[10:12])
	r.SrcIP = data[12:16]
	r.DstIP = data[16:20]

	headerLen := int(r.IHL) * 4
	if len(data) > headerLen {
		r.Payload = data[headerLen:]
	}

	return nil
}
