/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding/binary"
	"net"
)

type PortModificationMessage struct {
	Header
	Number    uint16
	MAC       net.HardwareAddr
	Config    PortConfig
	Advertise PortFeature
}

func (r *PortModificationMessage) MarshalBinary() ([]byte, error) {
	r.Header.Length = 32
	header, err := r.Header.MarshalBinary()
	if err != nil {
		return nil, err
	}

	v := make([]byte, r.Header.Length)
	copy(v[0:8], header)
	binary.BigEndian.PutUint16(v[8:10], r.Number)
	if r.MAC == nil || len(r.MAC) < 6 {
		copy(v[10:16], zeroMAC)
	} else {
		copy(v[10:16], r.MAC)
	}
	binary.BigEndian.PutUint32(v[16:20], uint32(r.Config))
	binary.BigEndian.PutUint32(v[20:24], 0xFFFFFFFF)
	binary.BigEndian.PutUint32(v[24:28], uint32(r.Advertise))

	return v, nil
}
