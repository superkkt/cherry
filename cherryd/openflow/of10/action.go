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

type Action struct {
	openflow.BaseAction
}

func marshalOutput(p uint) ([]byte, error) {
	v := make([]byte, 8)
	binary.BigEndian.PutUint16(v[0:2], uint16(OFPAT_OUTPUT))
	binary.BigEndian.PutUint16(v[2:4], 8)

	var port uint16
	switch p {
	case openflow.PortTable:
		port = OFPP_TABLE
	case openflow.PortAll:
		port = OFPP_ALL
	case openflow.PortController:
		port = OFPP_CONTROLLER
	case openflow.PortAny:
		port = OFPP_NONE
	default:
		port = uint16(p)
	}
	binary.BigEndian.PutUint16(v[4:6], port)
	// We don't support buffer ID and partial PACKET_IN
	binary.BigEndian.PutUint16(v[6:8], 0xFFFF)

	return v, nil
}

func marshalMAC(t uint16, mac net.HardwareAddr) ([]byte, error) {
	if mac == nil || len(mac) < 6 {
		return nil, errors.New("invalid MAC address")
	}

	v := make([]byte, 16)
	binary.BigEndian.PutUint16(v[0:2], t)
	binary.BigEndian.PutUint16(v[2:4], 16)
	copy(v[4:10], mac)

	return v, nil
}

func (r *Action) MarshalBinary() ([]byte, error) {
	result := make([]byte, 0)

	if ok, srcMAC := r.SrcMAC(); ok {
		v, err := marshalMAC(OFPAT_SET_DL_SRC, srcMAC)
		if err != nil {
			return nil, err
		}
		result = append(result, v...)
	}

	if ok, dstMAC := r.DstMAC(); ok {
		v, err := marshalMAC(OFPAT_SET_DL_DST, dstMAC)
		if err != nil {
			return nil, err
		}
		result = append(result, v...)
	}

	if ok, port := r.Output(); ok {
		v, err := marshalOutput(port)
		if err != nil {
			return nil, err
		}
		result = append(result, v...)
	}

	return result, nil
}

func (r *Action) UnmarshalBinary(data []byte) error {
	buf := data
	for len(buf) >= 4 {
		t := binary.BigEndian.Uint16(buf[0:2])
		length := binary.BigEndian.Uint16(buf[2:4])
		if len(buf) < int(length) {
			return openflow.ErrInvalidPacketLength
		}

		switch t {
		case OFPAT_OUTPUT:
			if len(buf) < 8 {
				return openflow.ErrInvalidPacketLength
			}
			if err := r.SetOutput(uint(binary.BigEndian.Uint16(buf[4:6]))); err != nil {
				return err
			}
		case OFPAT_SET_DL_SRC:
			if len(buf) < 16 {
				return openflow.ErrInvalidPacketLength
			}
			if err := r.SetSrcMAC(buf[4:10]); err != nil {
				return err
			}
		case OFPAT_SET_DL_DST:
			if len(buf) < 16 {
				return openflow.ErrInvalidPacketLength
			}
			if err := r.SetDstMAC(buf[4:10]); err != nil {
				return err
			}
		default:
			// Do nothing
		}

		buf = buf[length:]
	}

	return nil
}
