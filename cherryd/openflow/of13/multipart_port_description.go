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

type PortDescRequest struct {
	openflow.Message
}

func NewPortDescRequest(xid uint32) openflow.PortDescRequest {
	return &PortDescRequest{
		Message: openflow.NewMessage(openflow.OF13_VERSION, OFPT_MULTIPART_REQUEST, xid),
	}
}

func (r *PortDescRequest) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	// Multipart description request
	binary.BigEndian.PutUint16(v[0:2], OFPMP_PORT_DESC)
	// No flags and body
	r.SetPayload(v)

	return r.Message.MarshalBinary()
}

type PortDescReply struct {
	openflow.Message
	ports []openflow.Port
}

func (r PortDescReply) Ports() []openflow.Port {
	return r.ports
}

func (r *PortDescReply) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 8 {
		return openflow.ErrInvalidPacketLength
	}
	nPorts := (len(payload) - 8) / 64
	if nPorts == 0 {
		return nil
	}
	r.ports = make([]openflow.Port, nPorts)
	for i := 0; i < nPorts; i++ {
		buf := payload[8+i*64:]
		r.ports[i] = new(Port)
		if err := r.ports[i].UnmarshalBinary(buf[0:64]); err != nil {
			return err
		}
	}

	return nil
}
