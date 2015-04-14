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

type PortDescriptionRequest struct {
	openflow.Message
}

func NewPortDescriptionRequest(xid uint32) *PortDescriptionRequest {
	return &PortDescriptionRequest{
		Message: openflow.NewMessage(openflow.Ver13, OFPT_MULTIPART_REQUEST, xid),
	}
}

func (r *PortDescriptionRequest) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	// Multipart description request
	binary.BigEndian.PutUint16(v[0:2], OFPMP_PORT_DESC)
	// No flags and body
	r.SetPayload(v)

	return r.Message.MarshalBinary()
}

type PortDescriptionReply struct {
	openflow.Message
	Ports []*Port
}

func (r *PortDescriptionReply) UnmarshalBinary(data []byte) error {
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
	r.Ports = make([]*Port, nPorts)
	for i := 0; i < nPorts; i++ {
		buf := payload[8+i*64:]
		r.Ports[i] = new(Port)
		if err := r.Ports[i].UnmarshalBinary(buf[0:64]); err != nil {
			return err
		}
	}

	return nil
}
