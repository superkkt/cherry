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
	header openflow.Header
}

func NewPortDescriptionRequest(xid uint32) *PortDescriptionRequest {
	return &PortDescriptionRequest{
		header: openflow.Header{
			Version: openflow.Ver13,
			Type:    OFPT_MULTIPART_REQUEST,
			XID:     xid,
		},
	}
}

func (r *PortDescriptionRequest) Header() openflow.Header {
	return r.header
}

func (r *PortDescriptionRequest) MarshalBinary() ([]byte, error) {
	r.header.Length = 16
	header, err := r.header.MarshalBinary()
	if err != nil {
		return nil, err
	}

	v := make([]byte, r.header.Length)
	copy(v[0:8], header)
	binary.BigEndian.PutUint16(v[8:10], OFPMP_PORT_DESC)

	return v, nil
}

func (r *PortDescriptionRequest) UnmarshalBinary(data []byte) error {
	return openflow.ErrUnsupportedUnmarshaling
}

type PortDescriptionReply struct {
	header openflow.Header
	Ports  []*Port
}

func (r *PortDescriptionReply) Header() openflow.Header {
	return r.header
}

func (r *PortDescriptionReply) MarshalBinary() ([]byte, error) {
	return nil, openflow.ErrUnsupportedMarshaling
}

func (r *PortDescriptionReply) UnmarshalBinary(data []byte) error {
	if err := r.header.UnmarshalBinary(data); err != nil {
		return err
	}
	nPorts := (r.header.Length - 16) / 64
	if nPorts == 0 {
		return nil
	}
	r.Ports = make([]*Port, nPorts)
	for i := uint16(0); i < nPorts; i++ {
		buf := data[16+i*64:]
		r.Ports[i] = new(Port)
		if err := r.Ports[i].UnmarshalBinary(buf[0:64]); err != nil {
			return err
		}
	}

	return nil
}
