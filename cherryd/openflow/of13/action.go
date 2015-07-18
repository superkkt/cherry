/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service, Inc. All rights reserved.
 * Kitae Kim <superkkt@sds.co.kr>
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along
 * with this program; if not, write to the Free Software Foundation, Inc.,
 * 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
 */

package of13

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/superkkt/cherry/cherryd/openflow"
	"net"
)

type Action struct {
	*openflow.BaseAction
}

func NewAction() openflow.Action {
	return &Action{
		openflow.NewBaseAction(),
	}
}

func marshalOutput(p openflow.OutPort) ([]byte, error) {
	v := make([]byte, 16)
	binary.BigEndian.PutUint16(v[0:2], uint16(OFPAT_OUTPUT))
	binary.BigEndian.PutUint16(v[2:4], 16)

	var port uint32
	switch {
	case p.IsTable():
		port = OFPP_TABLE
	case p.IsFlood():
		port = OFPP_FLOOD
	case p.IsAll():
		port = OFPP_ALL
	case p.IsController():
		port = OFPP_CONTROLLER
	case p.IsNone():
		port = OFPP_ANY
	default:
		port = p.Value()
	}
	binary.BigEndian.PutUint32(v[4:8], port)
	// We don't support buffer ID and partial PACKET_IN
	binary.BigEndian.PutUint16(v[8:10], 0xFFFF)

	return v, nil
}

func marshalMAC(t uint8, mac net.HardwareAddr) ([]byte, error) {
	if mac == nil || len(mac) < 6 {
		return nil, openflow.ErrInvalidMACAddress
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

// TODO: Marshal Enqueue
func (r *Action) MarshalBinary() ([]byte, error) {
	if err := r.Error(); err != nil {
		return nil, err
	}

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

	v, err := marshalOutput(r.OutPort())
	if err != nil {
		return nil, err
	}
	result = append(result, v...)

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
			outPort := openflow.NewOutPort()
			outPort.SetValue(binary.BigEndian.Uint32(buf[4:8]))
			r.SetOutPort(outPort)
			if err := r.Error(); err != nil {
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
				r.SetDstMAC(buf[8:14])
				if err := r.Error(); err != nil {
					return err
				}
			case OFPXMT_OFB_ETH_SRC:
				if len(buf) < 14 {
					return openflow.ErrInvalidPacketLength
				}
				r.SetSrcMAC(buf[8:14])
				if err := r.Error(); err != nil {
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
