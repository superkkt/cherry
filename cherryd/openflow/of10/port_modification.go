/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of10

import (
	"encoding/binary"
	"errors"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"net"
)

type PortModification struct {
	openflow.Message
	number    uint16
	mac       net.HardwareAddr
	config    uint32
	advertise uint32
}

func NewPortModification(xid uint32, num uint16, mac net.HardwareAddr, config, advertise uint32) *PortModification {
	if mac == nil || len(mac) < 6 {
		panic("invalid MAC address!")
	}

	return &PortModification{
		Message:   openflow.NewMessage(openflow.Ver10, OFPT_PORT_MOD, xid),
		number:    num,
		mac:       mac,
		config:    config,
		advertise: advertise,
	}
}

func (r *PortModification) MarshalBinary() ([]byte, error) {
	v := make([]byte, 24)
	binary.BigEndian.PutUint16(v[0:2], r.number)
	if r.mac == nil || len(r.mac) < 6 {
		return nil, errors.New("invalid MAC address")
	} else {
		copy(v[2:8], r.mac)
	}
	binary.BigEndian.PutUint32(v[8:12], uint32(r.config))
	// Mask to set all bits from config
	binary.BigEndian.PutUint32(v[12:16], 0xFFFFFFFF)
	binary.BigEndian.PutUint32(v[16:20], uint32(r.advertise))
	// v[20:24] is padding
	r.SetPayload(v)

	return r.Message.MarshalBinary()
}
