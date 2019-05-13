/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015-2019 Samjung Data Service, Inc. All rights reserved.
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
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// See RFC 2131: Dynamic Host Configuration Protocol.
type DHCP struct {
	Op      DHCPOpcode // Message op code / message type.
	Hops    uint8
	XID     uint32        // Transaction ID.
	Elapsed time.Duration // Elapsed since client began address acquisition or renewal process.
	Flags   uint16
	CIAddr  net.IP           // Client IP address.
	YIAddr  net.IP           // Your (client) IP address.
	SIAddr  net.IP           // IP address of next server to use in bootstrap.
	GIAddr  net.IP           // Relay agent IP address, used in booting via a relay agent.
	CHAddr  net.HardwareAddr // Client hardware address.
	SName   string           // Optional server host name.
	File    string           // Boot file name.
	Options []DHCPOption     // Optional parameters field.

	m sync.Map // Map for options to search.
}

type DHCPOpcode uint8

const (
	DHCPOpcodeRequest DHCPOpcode = 1
	DHCPOpcodeReply   DHCPOpcode = 2
)

func (r *DHCP) MarshalBinary() ([]byte, error) {
	if r.Op != DHCPOpcodeRequest && r.Op != DHCPOpcodeReply {
		return nil, fmt.Errorf("invalid Op value: %v", r.Op)
	}

	v := make([]byte, 236)
	v[0] = byte(r.Op)
	v[1] = 1 // HType for Ethernet.
	v[2] = 6 // HLen for Ethernet.
	v[3] = r.Hops
	binary.BigEndian.PutUint32(v[4:8], r.XID)
	binary.BigEndian.PutUint16(v[8:10], uint16(r.Elapsed.Seconds()))
	binary.BigEndian.PutUint16(v[10:12], r.Flags)
	if err := marshalIP(v[12:16], r.CIAddr); err != nil {
		return nil, fmt.Errorf("CIAddr: %v", err)
	}
	if err := marshalIP(v[16:20], r.YIAddr); err != nil {
		return nil, fmt.Errorf("YIAddr: %v", err)
	}
	if err := marshalIP(v[20:24], r.SIAddr); err != nil {
		return nil, fmt.Errorf("SIAddr: %v", err)
	}
	if err := marshalIP(v[24:28], r.GIAddr); err != nil {
		return nil, fmt.Errorf("GIAddr: %v", err)
	}
	if err := marshalMAC(v[28:44], r.CHAddr); err != nil {
		return nil, fmt.Errorf("CHAddr: %v", err)
	}
	copy(v[44:107], r.SName)
	// v[108] = 0x0 for null-terminated string.
	copy(v[108:235], r.File)
	// v[236] = 0x0 for null-terminated string.

	// Magic cookie.
	v = append(v, []byte{0x63, 0x82, 0x53, 0x63}...)

	for _, opt := range r.Options {
		clv, err := opt.MarshalBinary()
		if err != nil {
			return nil, err
		}
		v = append(v, clv...)
	}
	v = append(v, byte(255)) // End option mark.

	// From RFC 2131: A DHCP client must be prepared to receive DHCP messages with an 'options' field of at least length 312 octets.
	const minDHCPPacketLen = 312
	if len(v) < minDHCPPacketLen {
		v = append(v, bytes.Repeat([]byte{0}, minDHCPPacketLen-len(v))...) // Zero padding.
	}

	return v, nil
}

func (r *DHCP) UnmarshalBinary(data []byte) (err error) {
	if len(data) < 236 {
		return errors.New("invalid DHCP packet length")
	}

	r.Op = DHCPOpcode(data[0])
	if r.Op != DHCPOpcodeRequest && r.Op != DHCPOpcodeReply {
		return fmt.Errorf("unexpected DHCP Opcode: %v", r.Op)
	}
	if data[1] != 1 || data[2] != 6 {
		return fmt.Errorf("unsupported hardware address type: HType=%v, HLen=%v", data[1], data[2])
	}
	r.Hops = data[3]
	r.XID = binary.BigEndian.Uint32(data[4:8])
	r.Elapsed = time.Duration(binary.BigEndian.Uint16(data[8:10])) * time.Second
	r.Flags = binary.BigEndian.Uint16(data[10:12])
	r.CIAddr = net.IPv4(data[12], data[13], data[14], data[15])
	r.YIAddr = net.IPv4(data[16], data[17], data[18], data[19])
	r.SIAddr = net.IPv4(data[20], data[21], data[22], data[23])
	r.GIAddr = net.IPv4(data[24], data[25], data[26], data[27])
	r.CHAddr, err = unmarshalMAC(data[28:44])
	if err != nil {
		return err
	}
	r.SName = unmarshalCString(data[44:108])
	r.File = unmarshalCString(data[108:236])

	// No options?
	if len(data[236:]) == 0 {
		return nil
	}

	opt := data[236:]
	for len(opt) > 0 {
		// End mark?
		if opt[0] == 255 {
			break
		}
		// Padding?
		if opt[0] == 0 {
			opt = opt[1:]
			continue
		}
		// Magic cookie?
		if opt[0] == 0x63 {
			if len(opt) >= 4 && opt[1] == 0x82 && opt[2] == 0x53 && opt[3] == 0x63 {
				opt = opt[4:]
				continue
			}
		}

		clv := DHCPOption{}
		if err := clv.UnmarshalBinary(opt); err != nil {
			return err
		}
		r.Options = append(r.Options, clv)
		r.m.Store(clv.Code, clv)

		opt = opt[2+len(clv.Value):] // +2 for Code and Length fields.
	}

	return nil
}

func (r *DHCP) Option(code uint8) (opt DHCPOption, ok bool) {
	v, ok := r.m.Load(code)
	if ok == false {
		return DHCPOption{}, false
	}

	return v.(DHCPOption), true
}

func unmarshalCString(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	idx := strings.Index(string(data), "\x00")
	if idx < 0 {
		// No null-terminated, just return the whole string.
		return string(data)
	}

	return string(data[:idx])
}

func marshalIP(v []byte, addr net.IP) error {
	if len(v) < 4 {
		panic("invalid destination buffer length")
	}
	if addr == nil {
		copy(v, []byte{0, 0, 0, 0})
		return nil
	}

	ipv4 := addr.To4()
	if ipv4 == nil {
		return fmt.Errorf("unsuppoted IP address type: %v", addr)
	}
	copy(v, ipv4)

	return nil
}

func marshalMAC(v []byte, mac net.HardwareAddr) error {
	if len(v) < 6 {
		panic("invalid destination buffer length")
	}
	if mac == nil {
		copy(v, []byte{0, 0, 0, 0, 0, 0})
		return nil
	}

	if len(mac) != 6 {
		return fmt.Errorf("unsupported MAC address type: %v", mac)
	}
	copy(v, mac)

	return nil
}

func unmarshalMAC(data []byte) (net.HardwareAddr, error) {
	if len(data) < 6 {
		return nil, errors.New("invalid MAC address length")
	}

	hw := make([]byte, 6)
	copy(hw, data)

	return hw, nil
}

// See RFC 2132: DHCP Options and BOOTP Vendor Extensions.
type DHCPOption struct {
	Code  uint8
	Value []byte
}

func (r *DHCPOption) MarshalBinary() ([]byte, error) {
	length := len(r.Value)
	if length == 0 || length > 255 {
		return nil, fmt.Errorf("invalid DHCP option packet length: code=%v, length=%v", r.Code, length)
	}

	v := make([]byte, 2+length)
	v[0] = r.Code
	v[1] = uint8(length)
	copy(v[2:], r.Value)

	return v, nil
}

func (r *DHCPOption) UnmarshalBinary(data []byte) error {
	if len(data) < 2 {
		return errors.New("invalid DHCP option packet header")
	}

	r.Code = data[0]
	length := uint8(data[1])
	if length == 0 || len(data)-2 < int(length) {
		return fmt.Errorf("invalid DHCP option packet length: code=%v, length=%v", r.Code, length)
	}
	r.Value = data[2 : 2+length]

	return nil
}
