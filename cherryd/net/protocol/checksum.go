/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
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
