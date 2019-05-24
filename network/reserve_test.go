/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015-2019 Samjung Data Service, Inc. All rights reserved.
 *  Kitae Kim <superkkt@sds.co.kr>
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

package network

import (
	"net"
	"testing"
)

func TestReservedIP(t *testing.T) {
	src := []struct {
		Network       net.IPNet
		Reserved      net.IP
		ErrorExpected bool
	}{
		{
			Network:       net.IPNet{IP: net.IPv4(223, 130, 121, 0), Mask: net.CIDRMask(24, 32)},
			Reserved:      net.IPv4(223, 130, 121, 188),
			ErrorExpected: false,
		},
		{
			Network:       net.IPNet{IP: net.IPv4(192, 168, 126, 100), Mask: net.CIDRMask(24, 32)},
			Reserved:      net.IPv4(192, 168, 126, 188),
			ErrorExpected: false,
		},
		{
			Network:       net.IPNet{IP: net.IPv4(192, 168, 126, 100), Mask: net.CIDRMask(25, 32)},
			Reserved:      net.IPv4(192, 168, 126, 126),
			ErrorExpected: false,
		},
		{
			Network:       net.IPNet{IP: net.IPv4(192, 168, 126, 250), Mask: net.CIDRMask(25, 32)},
			Reserved:      net.IPv4(192, 168, 126, 254),
			ErrorExpected: false,
		},
		{
			Network:       net.IPNet{IP: net.IPv4(10, 0, 0, 1), Mask: net.CIDRMask(8, 32)},
			Reserved:      net.IPv4(10, 255, 255, 254),
			ErrorExpected: false,
		},
		{
			Network:       net.IPNet{IP: net.IPv4(10, 0, 0, 1), Mask: net.CIDRMask(30, 32)},
			Reserved:      net.IPv4(10, 0, 0, 2),
			ErrorExpected: false,
		},
		{
			Network:       net.IPNet{IP: nil, Mask: net.CIDRMask(24, 32)},
			ErrorExpected: true,
		},
		{
			Network:       net.IPNet{IP: net.IPv4(10, 0, 0, 1), Mask: nil},
			ErrorExpected: true,
		},
		{
			Network:       net.IPNet{IP: nil, Mask: nil},
			ErrorExpected: true,
		},
		{
			Network:       net.IPNet{IP: net.IPv6zero, Mask: net.CIDRMask(24, 64)},
			ErrorExpected: true,
		},
		{
			Network:       net.IPNet{IP: net.IPv4(10, 0, 0, 1), Mask: net.CIDRMask(31, 32)},
			ErrorExpected: true,
		},
	}

	for _, v := range src {
		ip, err := ReservedIP(v.Network)
		if v.ErrorExpected == false && err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.ErrorExpected == true && err == nil {
			t.Fatal("expected error, but no error returns")
		}
		if ip.Equal(v.Reserved) == false {
			t.Fatalf("unexpected reserved IP address: expected=%v, actual=%v", v.Reserved, ip)
		}
	}
}
