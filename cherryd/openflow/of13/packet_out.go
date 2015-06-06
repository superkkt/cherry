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
)

type PacketOut struct {
	openflow.Message
	inPort openflow.InPort
	action openflow.Action
	data   []byte
}

func NewPacketOut(xid uint32) openflow.PacketOut {
	return &PacketOut{
		Message: openflow.NewMessage(openflow.OF13_VERSION, OFPT_PACKET_OUT, xid),
	}
}

func (r PacketOut) InPort() openflow.InPort {
	return r.inPort
}

func (r *PacketOut) SetInPort(port openflow.InPort) error {
	r.inPort = port
	return nil
}

func (r PacketOut) Action() openflow.Action {
	return r.action
}

func (r *PacketOut) SetAction(action openflow.Action) error {
	r.action = action
	return nil
}

func (r PacketOut) Data() []byte {
	return r.data
}

func (r *PacketOut) SetData(data []byte) error {
	r.data = data
	return nil
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

	v := make([]byte, 16)
	binary.BigEndian.PutUint32(v[0:4], OFP_NO_BUFFER)
	port := r.inPort.Port()
	if r.inPort.IsController() {
		port = OFPP_CONTROLLER
	}
	binary.BigEndian.PutUint32(v[4:8], port)
	binary.BigEndian.PutUint16(v[8:10], uint16(len(action)))
	// v[10:16] is padding
	v = append(v, action...)
	if r.data != nil && len(r.data) > 0 {
		v = append(v, r.data...)
	}

	r.SetPayload(v)
	return r.Message.MarshalBinary()
}
