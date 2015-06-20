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
)

func aroundCarry(sum uint32) uint32 {
	v := sum
	for {
		if (v >> 16) == 0 {
			break
		}
		upper := (v >> 16) & 0xFFFF
		lower := v & 0xFFFF
		v = upper + lower
	}

	return v
}

func calculateChecksum(header []byte) uint16 {
	v := header
	if len(v)%2 != 0 {
		v = append(v, byte(0))
	}

	var sum uint32 = 0
	for i := 0; i < len(v); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(v[i : i+2]))
	}
	sum = aroundCarry(sum)

	return ^uint16(sum)
}
