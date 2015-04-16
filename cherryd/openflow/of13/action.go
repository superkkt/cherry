/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

import (
	"bytes"
	"encoding/binary"
	"errors"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"net"
)

type Action struct {
	openflow.BaseAction
}

func marshalOutput(p uint) ([]byte, error) {
	v := make([]byte, 16)
	binary.BigEndian.PutUint16(v[0:2], uint16(OFPAT_OUTPUT))
	binary.BigEndian.PutUint16(v[2:4], 16)

	var port uint32
	switch p {
	case openflow.PortTable:
		port = OFPP_TABLE
	case openflow.PortAll:
		port = OFPP_ALL
	case openflow.PortController:
		port = OFPP_CONTROLLER
	case openflow.PortAny:
		port = OFPP_ANY
	default:
		port = uint32(p)
	}
	binary.BigEndian.PutUint32(v[4:8], port)
	// We don't support buffer ID and partial PACKET_IN
	binary.BigEndian.PutUint16(v[8:10], 0xFFFF)

	return v, nil
}

func marshalMAC(t uint8, mac net.HardwareAddr) ([]byte, error) {
	if mac == nil || len(mac) < 6 {
		return nil, errors.New("invalid MAC address")
	}

	tlv, err := marshalHardwareAddrTLV(t, mac)
	if err != nil {
		return nil, err
	}

	v := make([]byte, 4+len(tlv))
	binary.BigEndian.PutUint16(v[0:2], OFPAT_SET_FIELD)
	// Add padding to align as a multiple of 8
	rem := (len(v)) % 8
	if rem > 0 {
		v = append(v, bytes.Repeat([]byte{0}, 8-rem)...)
	}
	binary.BigEndian.PutUint16(v[2:4], uint16(len(v)))
	copy(v[4:], tlv)

	return v, nil
}

func (r *Action) MarshalBinary() ([]byte, error) {
	result := make([]byte, 0)

	if ok, srcMAC := r.SrcMAC(); ok {
		v, err := marshalMAC(OFPXMT_OFB_ETH_SRC, srcMAC)
		if err != nil {
			return nil, err
		}
		result = append(result, v...)
	}

	if ok, dstMAC := r.DstMAC(); ok {
		v, err := marshalMAC(OFPXMT_OFB_ETH_DST, dstMAC)
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
			if err := r.SetOutput(uint(binary.BigEndian.Uint32(buf[4:8]))); err != nil {
				return err
			}
		case OFPAT_SET_FIELD:
			if len(buf) < 8 {
				return openflow.ErrInvalidPacketLength
			}
			header := binary.BigEndian.Uint32(buf[4:8])
			class := header >> 16 & 0xFFFF
			if class != 0x8000 {
				return errors.New("unsupported TLV class")
			}
			field := header >> 9 & 0x7F

			switch field {
			case OFPXMT_OFB_ETH_DST:
				if len(buf) < 14 {
					return openflow.ErrInvalidPacketLength
				}
				if err := r.SetDstMAC(buf[8:14]); err != nil {
					return err
				}
			case OFPXMT_OFB_ETH_SRC:
				if len(buf) < 14 {
					return openflow.ErrInvalidPacketLength
				}
				if err := r.SetSrcMAC(buf[8:14]); err != nil {
					return err
				}
			default:
				// Do nothing
			}
		default:
			// Do nothing
		}

		buf = buf[length:]
	}

	return nil
}
