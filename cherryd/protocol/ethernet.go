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

type Ethernet struct {
	SrcMAC, DstMAC net.HardwareAddr
	Type           uint16
	Payload        []byte
}

func (r Ethernet) MarshalBinary() ([]byte, error) {
	if r.SrcMAC == nil || r.DstMAC == nil {
		return nil, errors.New("invalid MAC address")
	}
	if r.Payload == nil {
		return nil, errors.New("nil payload")
	}

	v := make([]byte, 14+len(r.Payload))
	copy(v[0:6], r.DstMAC)
	copy(v[6:12], r.SrcMAC)
	binary.BigEndian.PutUint16(v[12:14], r.Type)
	if len(r.Payload) > 0 {
		copy(v[14:], r.Payload)
	}

	return v, nil
}

func (r *Ethernet) UnmarshalBinary(data []byte) error {
	if len(data) < 14 {
		return errors.New("invalid ethernet frame length")
	}

	r.DstMAC = data[0:6]
	r.SrcMAC = data[6:12]
	r.Type = binary.BigEndian.Uint16(data[12:14])
	// IEEE 802.1Q-tagged frame?
	if r.Type == 0x8100 {
		r.Type = binary.BigEndian.Uint16(data[16:18])
		r.Payload = data[18:]
	} else {
		r.Payload = data[14:]
	}
	// FIXME: Add routines for JumboFrame

	return nil
}
