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

package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

type UDP struct {
	srcIP    net.IP
	dstIP    net.IP
	SrcPort  uint16
	DstPort  uint16
	Length   uint16
	Checksum uint16
	Payload  []byte
}

// UDP checksum needs a pseudo header that has src and dst IPv4 addresses.
func (r *UDP) SetPseudoHeader(src, dst net.IP) {
	r.srcIP = src
	r.dstIP = dst
}

func (r UDP) MarshalBinary() ([]byte, error) {
	length := 8
	if r.Payload != nil {
		length += len(r.Payload)
	}
	if length > (0xFFFF - 20 /* IPv4 header */) {
		return nil, fmt.Errorf("too long UDP packet: length=%v", length)
	}

	v := make([]byte, length)
	binary.BigEndian.PutUint16(v[0:2], r.SrcPort)
	binary.BigEndian.PutUint16(v[2:4], r.DstPort)
	binary.BigEndian.PutUint16(v[4:6], uint16(length))
	// v[6:8] is checksum
	if r.Payload != nil && len(r.Payload) > 0 {
		copy(v[8:], r.Payload)
	}

	if r.srcIP == nil || r.dstIP == nil {
		return nil, errors.New("nil pseudo IP addresses")
	}
	pseudo := make([]byte, 12)
	srcIP := r.srcIP.To4()
	if srcIP == nil {
		return nil, errors.New("source IP address is not an IPv4 address")
	}
	copy(pseudo[0:4], srcIP)
	dstIP := r.dstIP.To4()
	if dstIP == nil {
		return nil, errors.New("destination IP address is not an IPv4 address")
	}
	copy(pseudo[4:8], dstIP)
	pseudo[9] = 17 // UDP
	binary.BigEndian.PutUint16(pseudo[10:12], uint16(length))

	checksum := calculateChecksum(append(pseudo, v...))
	binary.BigEndian.PutUint16(v[6:8], checksum)

	return v, nil
}

func (r *UDP) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return errors.New("invalid UDP packet length")
	}

	r.SrcPort = binary.BigEndian.Uint16(data[0:2])
	r.DstPort = binary.BigEndian.Uint16(data[2:4])
	r.Length = binary.BigEndian.Uint16(data[4:6])
	r.Checksum = binary.BigEndian.Uint16(data[6:8])
	if len(data) > 8 {
		r.Payload = data[8:]
	}

	return nil
}
