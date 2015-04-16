/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of10

import (
	"encoding/binary"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type PacketOut struct {
	openflow.Message
	inPort uint16
	action openflow.Action
	data   []byte
}

func NewPacketOut(xid uint32, inport openflow.InPort, action openflow.Action, data []byte) *PacketOut {
	port := uint16(inport.Port())
	if inport.IsController() {
		port = OFPP_CONTROLLER
	}

	return &PacketOut{
		Message: openflow.NewMessage(openflow.Ver10, OFPT_PACKET_OUT, xid),
		inPort:  port,
		action:  action,
		data:    data,
	}
}

func (r *PacketOut) MarshalBinary() ([]byte, error) {
	action := make([]byte, 0)
	if r.action != nil {
		a, err := r.action.MarshalBinary()
		if err != nil {
			return nil, err
		}
		action = append(action, a...)
	}

	v := make([]byte, 8)
	binary.BigEndian.PutUint32(v[0:4], OFP_NO_BUFFER)
	binary.BigEndian.PutUint16(v[4:6], r.inPort)
	binary.BigEndian.PutUint16(v[6:8], uint16(len(action)))
	v = append(v, action...)
	if r.data != nil && len(r.data) > 0 {
		v = append(v, r.data...)
	}

	r.SetPayload(v)
	return r.Message.MarshalBinary()
}
