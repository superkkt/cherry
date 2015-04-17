/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

import (
	"encoding/binary"
	"errors"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"net"
)

// XXX: PORT_MOD does not result in PORT_STATUS issued from the switch

type PortModification struct {
	openflow.Message
	number    uint32
	mac       net.HardwareAddr
	config    uint32
	advertise uint32
}

func NewPortModification(xid, num uint32, mac net.HardwareAddr, config, advertise uint32) *PortModification {
	if mac == nil || len(mac) < 6 {
		panic("invalid MAC address!")
	}

	return &PortModification{
		Message:   openflow.NewMessage(openflow.Ver13, OFPT_PORT_MOD, xid),
		number:    num,
		mac:       mac,
		config:    config,
		advertise: advertise,
	}
}

func (r *PortModification) MarshalBinary() ([]byte, error) {
	v := make([]byte, 32)
	binary.BigEndian.PutUint32(v[0:4], r.number)
	// v[4:8] is padding
	if r.mac == nil || len(r.mac) < 6 {
		return nil, errors.New("invalid MAC address")
	} else {
		copy(v[8:14], r.mac)
	}
	// v[14:16] is padding
	binary.BigEndian.PutUint32(v[16:20], uint32(r.config))
	// Mask to set all bits from config
	binary.BigEndian.PutUint32(v[20:24], 0xFFFFFFFF)
	binary.BigEndian.PutUint32(v[24:28], uint32(r.advertise))
	// v[28:32] is padding
	r.SetPayload(v)

	return r.Message.MarshalBinary()
}
