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
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestCodec(t *testing.T) {
	samples := []struct {
		Packet     string
		Expected   DHCP
		DecodeOnly bool
	}{
		{
			Packet: "0101060000003d1d0000000000000000000000000000000000000000000b8201fc4200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000638253633501013d0701000b8201fc4232040000000037040103062aff0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			Expected: DHCP{
				Op:      DHCPOpcodeRequest,
				Hops:    0,
				XID:     0x3d1d,
				Elapsed: 0,
				Flags:   0,
				CIAddr:  net.IPv4zero,
				YIAddr:  net.IPv4zero,
				SIAddr:  net.IPv4zero,
				GIAddr:  net.IPv4zero,
				CHAddr:  []byte{0x00, 0x0b, 0x82, 0x01, 0xfc, 0x42},
				SName:   "",
				File:    "",
				Options: []DHCPOption{
					{Code: 0x35, Value: []byte{0x01}},
					{Code: 0x3d, Value: []byte{0x01, 0x00, 0x0b, 0x82, 0x01, 0xfc, 0x42}},
					{Code: 0x32, Value: []byte{0x00, 0x00, 0x00, 0x00}},
					{Code: 0x37, Value: []byte{0x01, 0x03, 0x06, 0x2a}},
				},
			},
			DecodeOnly: false,
		},
		{
			Packet: "0201060000003d1d0000000000000000c0a8000ac0a8000100000000000b8201fc4200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000638253633501020104ffffff003a04000007083b0400000c4e330400000e103604c0a80001ff0000000000000000000000000000000000000000000000000000000000000000000000000000",
			Expected: DHCP{
				Op:      DHCPOpcodeReply,
				Hops:    0,
				XID:     0x3d1d,
				Elapsed: 0,
				Flags:   0,
				CIAddr:  net.IPv4zero,
				YIAddr:  net.IPv4(192, 168, 0, 10),
				SIAddr:  net.IPv4(192, 168, 0, 1),
				GIAddr:  net.IPv4zero,
				CHAddr:  []byte{0x00, 0x0b, 0x82, 0x01, 0xfc, 0x42},
				SName:   "",
				File:    "",
				Options: []DHCPOption{
					{Code: 0x35, Value: []byte{0x02}},
					{Code: 0x01, Value: []byte{0xff, 0xff, 0xff, 0x00}},
					{Code: 0x3a, Value: []byte{0x00, 0x00, 0x07, 0x08}},
					{Code: 0x3b, Value: []byte{0x00, 0x00, 0x0c, 0x4e}},
					{Code: 0x33, Value: []byte{0x00, 0x00, 0x0e, 0x10}},
					{Code: 0x36, Value: []byte{0xc0, 0xa8, 0x00, 0x01}},
				},
			},
			DecodeOnly: false,
		},
		{
			Packet: "0101060000003d1e0000000000000000000000000000000000000000000b8201fc4200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000638253633501033d0701000b8201fc423204c0a8000a3604c0a8000137040103062aff0000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			Expected: DHCP{
				Op:      DHCPOpcodeRequest,
				Hops:    0,
				XID:     0x3d1e,
				Elapsed: 0,
				Flags:   0,
				CIAddr:  net.IPv4zero,
				YIAddr:  net.IPv4zero,
				SIAddr:  net.IPv4zero,
				GIAddr:  net.IPv4zero,
				CHAddr:  []byte{0x00, 0x0b, 0x82, 0x01, 0xfc, 0x42},
				SName:   "",
				File:    "",
				Options: []DHCPOption{
					{Code: 0x35, Value: []byte{0x03}},
					{Code: 0x3d, Value: []byte{0x01, 0x00, 0x0b, 0x82, 0x01, 0xfc, 0x42}},
					{Code: 0x32, Value: []byte{0xc0, 0xa8, 0x00, 0x0a}},
					{Code: 0x36, Value: []byte{0xc0, 0xa8, 0x00, 0x01}},
					{Code: 0x37, Value: []byte{0x01, 0x03, 0x06, 0x2a}},
				},
			},
			DecodeOnly: false,
		},
		{
			Packet: "0201060000003d1e0000000000000000c0a8000a0000000000000000000b8201fc4200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000638253633501053a04000007083b0400000c4e330400000e103604c0a800010104ffffff00ff0000000000000000000000000000000000000000000000000000000000000000000000000000",
			Expected: DHCP{
				Op:      DHCPOpcodeReply,
				Hops:    0,
				XID:     0x3d1e,
				Elapsed: 0,
				Flags:   0,
				CIAddr:  net.IPv4zero,
				YIAddr:  net.IPv4(192, 168, 0, 10),
				SIAddr:  net.IPv4zero,
				GIAddr:  net.IPv4zero,
				CHAddr:  []byte{0x00, 0x0b, 0x82, 0x01, 0xfc, 0x42},
				SName:   "",
				File:    "",
				Options: []DHCPOption{
					{Code: 0x35, Value: []byte{0x05}},
					{Code: 0x3a, Value: []byte{0x00, 0x00, 0x07, 0x08}},
					{Code: 0x3b, Value: []byte{0x00, 0x00, 0x0c, 0x4e}},
					{Code: 0x33, Value: []byte{0x00, 0x00, 0x0e, 0x10}},
					{Code: 0x36, Value: []byte{0xc0, 0xa8, 0x00, 0x01}},
					{Code: 0x01, Value: []byte{0xff, 0xff, 0xff, 0x00}},
				},
			},
			DecodeOnly: false,
		},
		{
			Packet: "020106017771cf85000a0000000000000a0a08ebac16b2ea0a0a08f0000e8611c07500000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000638253633501020104ffffff003604ac16b2ea33040000a8c003040a0a08fe06088fd104018fd10501420e3137322e32322e3137382e323334780501ac16b2ea3d10006e617468616e31636c69656e7469645a1f010100c878c45256402081313233348fe0cce2ee8596abb25817c480b2fd305216011420504f4e20312f312f30372f30313a312e302e31ff",
			Expected: DHCP{
				Op:      DHCPOpcodeReply,
				Hops:    1,
				XID:     0x7771cf85,
				Elapsed: 10 * time.Second,
				Flags:   0,
				CIAddr:  net.IPv4zero,
				YIAddr:  net.IPv4(10, 10, 8, 235),
				SIAddr:  net.IPv4(172, 22, 178, 234),
				GIAddr:  net.IPv4(10, 10, 8, 240),
				CHAddr:  []byte{0x00, 0x0e, 0x86, 0x11, 0xc0, 0x75},
				SName:   "",
				File:    "",
				Options: []DHCPOption{
					{Code: 0x35, Value: []byte{0x02}},
					{Code: 0x01, Value: []byte{0xff, 0xff, 0xff, 0x00}},
					{Code: 0x36, Value: []byte{0xac, 0x16, 0xb2, 0xea}},
					{Code: 0x33, Value: []byte{0x00, 0x00, 0xa8, 0xc0}},
					{Code: 0x03, Value: []byte{0x0a, 0x0a, 0x08, 0xfe}},
					{Code: 0x06, Value: []byte{0x8f, 0xd1, 0x04, 0x01, 0x8f, 0xd1, 0x05, 0x01}},
					{Code: 0x42, Value: []byte{0x31, 0x37, 0x32, 0x2e, 0x32, 0x32, 0x2e, 0x31, 0x37, 0x38, 0x2e, 0x32, 0x33, 0x34}},
					{Code: 0x78, Value: []byte{0x01, 0xac, 0x16, 0xb2, 0xea}},
					{Code: 0x3d, Value: []byte{0x00, 0x6e, 0x61, 0x74, 0x68, 0x61, 0x6e, 0x31, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x69, 0x64}},
					{Code: 0x5a, Value: []byte{0x01, 0x01, 0x00, 0xc8, 0x78, 0xc4, 0x52, 0x56, 0x40, 0x20, 0x81, 0x31, 0x32, 0x33, 0x34, 0x8f, 0xe0, 0xcc, 0xe2, 0xee, 0x85, 0x96, 0xab, 0xb2, 0x58, 0x17, 0xc4, 0x80, 0xb2, 0xfd, 0x30}},
					{Code: 0x52, Value: []byte{0x01, 0x14, 0x20, 0x50, 0x4f, 0x4e, 0x20, 0x31, 0x2f, 0x31, 0x2f, 0x30, 0x37, 0x2f, 0x30, 0x31, 0x3a, 0x31, 0x2e, 0x30, 0x2e, 0x31}},
				},
			},
			DecodeOnly: false,
		},
		{
			// Unsupported option overload packet.
			Packet: "01010600ac2effff000000000000000000000000000000000000000000006c82dc4e000000000000000000003814736e616d65206669656c64206f7665726c6f6164ff0000000000000000000000000000000000000000000000000000000000000000000000000000000000381866696c65206e616d65206669656c64206f7665726c6f6164ff0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000638253633501013902024e3704011c032b330400000e10340103380750616464696e67003d070100006c82dc4eff",
			Expected: DHCP{
				Op:      DHCPOpcodeRequest,
				Hops:    0,
				XID:     0xac2effff,
				Elapsed: 0,
				Flags:   0,
				CIAddr:  net.IPv4zero,
				YIAddr:  net.IPv4zero,
				SIAddr:  net.IPv4zero,
				GIAddr:  net.IPv4zero,
				CHAddr:  []byte{0x00, 0x00, 0x6c, 0x82, 0xdc, 0x4e},
				SName:   "8\x14sname field overload\xff",
				File:    "8\x18file name field overload\xff",
				Options: []DHCPOption{
					{Code: 0x35, Value: []byte{0x01}},
					{Code: 0x39, Value: []byte{0x02, 0x4e}},
					{Code: 0x37, Value: []byte{0x01, 0x1c, 0x03, 0x2b}},
					{Code: 0x33, Value: []byte{0x00, 0x00, 0x0e, 0x10}},
					{Code: 0x34, Value: []byte{0x03}},
					{Code: 0x38, Value: []byte{0x50, 0x61, 0x64, 0x64, 0x69, 0x6e, 0x67}},
					{Code: 0x3d, Value: []byte{0x01, 0x00, 0x00, 0x6c, 0x82, 0xdc, 0x4e}},
				},
			},
			DecodeOnly: true,
		},
		{
			// Unsupported option overload packet without the end mark.
			Packet: "01010600ac2effff000000000000000000000000000000000000000000006c82dc4e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000638253633501013902024e3704011c032b330400000e10340103380750616464696e67003d070100006c82dc4e00",
			Expected: DHCP{
				Op:      DHCPOpcodeRequest,
				Hops:    0,
				XID:     0xac2effff,
				Elapsed: 0,
				Flags:   0,
				CIAddr:  net.IPv4zero,
				YIAddr:  net.IPv4zero,
				SIAddr:  net.IPv4zero,
				GIAddr:  net.IPv4zero,
				CHAddr:  []byte{0x00, 0x00, 0x6c, 0x82, 0xdc, 0x4e},
				SName:   "",
				File:    "",
				Options: []DHCPOption{
					{Code: 0x35, Value: []byte{0x01}},
					{Code: 0x39, Value: []byte{0x02, 0x4e}},
					{Code: 0x37, Value: []byte{0x01, 0x1c, 0x03, 0x2b}},
					{Code: 0x33, Value: []byte{0x00, 0x00, 0x0e, 0x10}},
					{Code: 0x34, Value: []byte{0x03}},
					{Code: 0x38, Value: []byte{0x50, 0x61, 0x64, 0x64, 0x69, 0x6e, 0x67}},
					{Code: 0x3d, Value: []byte{0x01, 0x00, 0x00, 0x6c, 0x82, 0xdc, 0x4e}},
				},
			},
			DecodeOnly: true,
		},
	}

	for _, v := range samples {
		p, err := hex.DecodeString(v.Packet)
		if err != nil {
			panic("invalid sample DHCP packet")
		}

		d := DHCP{}
		if err := d.UnmarshalBinary(p); err != nil {
			t.Fatalf("unexpected DHCP unmarshal error: %v", err)
		}
		if cmp.Equal(d, v.Expected, cmpopts.IgnoreUnexported(d, v.Expected)) == false {
			t.Fatalf("unexpected unmarshaled DHCP packet: expected=%v, actual=%v, diff=%v", spew.Sdump(v.Expected), spew.Sdump(d), cmp.Diff(d, v.Expected, cmpopts.IgnoreUnexported(d, v.Expected)))
		}

		if v.DecodeOnly == true {
			continue
		}

		m, err := d.MarshalBinary()
		if err != nil {
			t.Fatalf("unexpected DHCP marshal error: %v", err)
		}
		if bytes.Equal(m, p) == false {
			t.Fatalf("unexpected marshaled DHCP packet: expected=%v, actual=%v", p, m)
		}
	}
}
