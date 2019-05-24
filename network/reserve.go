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
	"encoding/binary"
	"fmt"
	"net"
)

// ReservedIP returns the IP address reserved for the network.
func ReservedIP(n net.IPNet) (net.IP, error) {
	if n.IP == nil || n.Mask == nil {
		return nil, fmt.Errorf("invalid IP network: %v", n)
	}

	ip := n.IP.Mask(n.Mask).To4()
	if ip == nil {
		return nil, fmt.Errorf("invalid IPv4 address: %v", n)
	}

	ones, bits := n.Mask.Size()
	if (ones == 0 && bits == 0) || bits <= ones || bits != 32 || ones > 30 {
		return nil, fmt.Errorf("invalid IP mask: %v", n)
	}

	if ones == 24 {
		// This is a workaround because x.x.x.254 IP addresses are already in use.
		return net.IPv4(ip[0], ip[1], ip[2], 188), nil
	}
	// Use the final IP address of the network.
	v := binary.BigEndian.Uint32(ip)
	h := uint32(1<<uint(bits-ones) - 2)
	binary.BigEndian.PutUint32(ip, v|h)

	return ip, nil
}
