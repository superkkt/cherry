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
