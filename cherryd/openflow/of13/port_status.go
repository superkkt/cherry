/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

import (
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type PortStatus struct {
	header openflow.Header
	Reason uint8
	Port   *Port
}

func (r *PortStatus) Header() openflow.Header {
	return r.header
}

func (r *PortStatus) MarshalBinary() ([]byte, error) {
	return nil, openflow.ErrUnsupportedMarshaling
}

func (r *PortStatus) UnmarshalBinary(data []byte) error {
	if err := r.header.UnmarshalBinary(data); err != nil {
		return err
	}
	if r.header.Length < 80 || len(data) < int(r.header.Length) {
		return openflow.ErrInvalidPacketLength
	}

	r.Reason = data[9]
	r.Port = new(Port)
	if err := r.Port.UnmarshalBinary(data[16:]); err != nil {
		return err
	}

	return nil
}
