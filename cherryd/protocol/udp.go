/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package protocol

import (
	"encoding/binary"
	"errors"
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

	v := make([]byte, length)
	binary.BigEndian.PutUint16(v[0:2], r.SrcPort)
	binary.BigEndian.PutUint16(v[2:4], r.DstPort)
	binary.BigEndian.PutUint16(v[4:6], r.Length)
	// v[6:8] is checksum
	if r.Payload != nil && len(r.Payload) > 0 {
		copy(v[8:], r.Payload)
	}

	if r.srcIP == nil || r.dstIP == nil {
		return nil, errors.New("nil pseudo IP addresses")
	}
	pseudo := make([]byte, 12)
	copy(pseudo[0:4], r.srcIP)
	copy(pseudo[4:8], r.dstIP)
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
