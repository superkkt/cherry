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
)

type ICMP struct {
	Type     uint8
	Code     uint8
	Checksum uint16
}

type ICMPEcho struct {
	ICMP
	ID       uint16
	Sequence uint16
	Payload  []byte
}

func NewICMPEchoRequest(id, seq uint16, payload []byte) *ICMPEcho {
	return &ICMPEcho{
		ICMP: ICMP{
			Type: 8,
		},
		ID:       id,
		Sequence: seq,
		Payload:  payload,
	}
}

func NewICMPEchoReply(id, seq uint16, payload []byte) *ICMPEcho {
	return &ICMPEcho{
		ID:       id,
		Sequence: seq,
		Payload:  payload,
	}
}

func (r ICMPEcho) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	v[0] = r.Type
	v[1] = r.Code
	// v[2:4] is checksum
	binary.BigEndian.PutUint16(v[4:6], r.ID)
	binary.BigEndian.PutUint16(v[6:8], r.Sequence)
	if r.Payload != nil {
		v = append(v, r.Payload...)
	}

	checksum := calculateChecksum(v)
	binary.BigEndian.PutUint16(v[2:4], checksum)

	return v, nil
}

func (r *ICMPEcho) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return errors.New("invalid ICMP packet length")
	}
	if data[0] != 8 && data[0] != 0 {
		return errors.New("packet is not an ICMP echo message")
	}

	r.Type = data[0]
	r.Code = data[1]
	r.Checksum = binary.BigEndian.Uint16(data[2:4])
	r.ID = binary.BigEndian.Uint16(data[4:6])
	r.Sequence = binary.BigEndian.Uint16(data[6:8])
	if len(data) > 8 {
		r.Payload = data[8:]
	}

	return nil
}
