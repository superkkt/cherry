/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

import (
	"encoding/binary"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"net"
	"strings"
)

type Port struct {
	number uint32
	mac    net.HardwareAddr
	name   string
	// Bitmap of OFPPC_* flags
	config uint32
	// Bitmap of OFPPS_* flags
	state uint32
	//
	//  Bitmaps of OFPPF_* that describe features. All bits zeroed if unsupported or unavailable.
	//
	current, advertised, supported, peer uint32
	currentSpeed, maxSpeed               uint32
}

func (r Port) Number() uint32 {
	return r.number
}

func (r Port) MAC() net.HardwareAddr {
	return r.mac
}

func (r Port) Name() string {
	return r.name
}

func (r Port) IsPortDown() bool {
	if r.config&OFPPC_PORT_DOWN != 0 {
		return true
	}

	return false
}

func (r Port) IsLinkDown() bool {
	if r.state&OFPPS_LINK_DOWN != 0 {
		return true
	}

	return false
}

func (r Port) IsCopper() bool {
	return r.current&OFPPF_COPPER != 0
}

func (r Port) IsFiber() bool {
	return r.current&OFPPF_FIBER != 0
}

func (r Port) IsAutoNego() bool {
	return r.current&OFPPF_AUTONEG != 0
}

func (r *Port) Speed() uint64 {
	switch {
	case r.current&OFPPF_10MB_HD != 0:
		return 5
	case r.current&OFPPF_10MB_FD != 0:
		return 10
	case r.current&OFPPF_100MB_HD != 0:
		return 50
	case r.current&OFPPF_100MB_FD != 0:
		return 100
	case r.current&OFPPF_1GB_HD != 0:
		return 500
	case r.current&OFPPF_1GB_FD != 0:
		return 1000
	case r.current&OFPPF_10GB_FD != 0:
		return 10000
	case r.current&OFPPF_40GB_FD != 0:
		return 40000
	case r.current&OFPPF_100GB_FD != 0:
		return 100000
	case r.current&OFPPF_1TB_FD != 0:
		return 1000000
	default:
		return 0
	}
}

func (r *Port) UnmarshalBinary(data []byte) error {
	if len(data) < 64 {
		return openflow.ErrInvalidPacketLength
	}

	r.number = binary.BigEndian.Uint32(data[0:4])
	r.mac = make(net.HardwareAddr, 6)
	copy(r.mac, data[8:14])
	r.name = strings.TrimRight(string(data[16:32]), "\x00")
	r.config = binary.BigEndian.Uint32(data[32:36])
	r.state = binary.BigEndian.Uint32(data[36:40])
	r.current = binary.BigEndian.Uint32(data[40:44])
	r.advertised = binary.BigEndian.Uint32(data[44:48])
	r.supported = binary.BigEndian.Uint32(data[48:52])
	r.peer = binary.BigEndian.Uint32(data[52:56])
	r.currentSpeed = binary.BigEndian.Uint32(data[56:60])
	r.maxSpeed = binary.BigEndian.Uint32(data[60:64])

	return nil
}
