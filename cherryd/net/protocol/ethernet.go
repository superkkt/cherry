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
	"hash/crc32"
	"net"
)

type Ethernet struct {
	SrcMAC, DstMAC net.HardwareAddr
	Type           uint16
	Payload        []byte
}

func (r *Ethernet) MarshalBinary() ([]byte, error) {
	if r.SrcMAC == nil || r.DstMAC == nil {
		return nil, errors.New("invalid MAC address")
	}
	if r.Payload == nil {
		return nil, errors.New("nil payload")
	}

	length := 64
	if len(r.Payload) > 46 {
		length += len(r.Payload) - 46
	}

	v := make([]byte, length)
	copy(v[0:6], r.DstMAC)
	copy(v[6:12], r.SrcMAC)
	binary.BigEndian.PutUint16(v[12:14], r.Type)
	copy(v[14:], r.Payload)
	binary.BigEndian.PutUint32(v[length-4:], crc32.ChecksumIEEE(v[0:length-4]))

	return v, nil
}

func (r *Ethernet) UnmarshalBinary(data []byte) error {
	length := len(data)
	if length < 14 {
		return errors.New("invalid ethernet frame length")
	}

	r.DstMAC = data[0:6]
	r.SrcMAC = data[6:12]
	r.Type = binary.BigEndian.Uint16(data[12:14])
	r.Payload = data[14:]

	return nil
}
