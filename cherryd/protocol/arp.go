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

type ARP struct {
	HWType      uint16
	ProtoType   uint16
	HWLength    uint8
	ProtoLength uint8
	Operation   uint16
	SHA         net.HardwareAddr // Sender Hardware Address
	SPA         net.IP           // Sender Protocol Address
	THA         net.HardwareAddr // Target Hardware Address
	TPA         net.IP           // Target Protocol Address
}

func NewARPRequest(sha net.HardwareAddr, spa, tpa net.IP) *ARP {
	return &ARP{
		HWType:      1,      // Ethernet
		ProtoType:   0x0800, // IPv4
		HWLength:    6,      // Size of Ethernet MAC address
		ProtoLength: 4,      // Size of IPv4 address
		Operation:   1,      // ARP request
		SHA:         sha,
		SPA:         spa,
		TPA:         tpa,
	}
}

func NewARPReply(sha, tha net.HardwareAddr, spa, tpa net.IP) *ARP {
	return &ARP{
		HWType:      1,      // Ethernet
		ProtoType:   0x0800, // IPv4
		HWLength:    6,      // Size of Ethernet MAC address
		ProtoLength: 4,      // Size of IPv4 address
		Operation:   2,      // ARP reply
		SHA:         sha,
		SPA:         spa,
		THA:         tha,
		TPA:         tpa,
	}
}

func (r ARP) MarshalBinary() ([]byte, error) {
	if r.SHA == nil || r.SPA == nil || r.THA == nil || r.TPA == nil {
		return nil, errors.New("invalid hardware or protocol address")
	}

	v := make([]byte, 28)
	binary.BigEndian.PutUint16(v[0:2], r.HWType)
	binary.BigEndian.PutUint16(v[2:4], r.ProtoType)
	v[4] = r.HWLength
	v[5] = r.ProtoLength
	binary.BigEndian.PutUint16(v[6:8], r.Operation)
	copy(v[8:14], r.SHA)
	copy(v[14:18], r.SPA)
	copy(v[18:24], r.THA)
	copy(v[24:28], r.TPA)

	return v, nil
}

func (r *ARP) UnmarshalBinary(data []byte) error {
	if len(data) < 28 {
		return errors.New("invalid ARP packet length")
	}

	r.HWType = binary.BigEndian.Uint16(data[0:2])
	r.ProtoType = binary.BigEndian.Uint16(data[2:4])
	r.HWLength = data[4]
	r.ProtoLength = data[5]
	r.Operation = binary.BigEndian.Uint16(data[6:8])
	r.SHA = data[8:14]
	r.SPA = data[14:18]
	r.THA = data[18:24]
	r.TPA = data[24:28]

	return nil
}
