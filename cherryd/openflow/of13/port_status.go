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
	openflow.Message
	Reason uint8
	Port   *Port
}

func (r *PortStatus) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 72 {
		return openflow.ErrInvalidPacketLength
	}
	r.Reason = payload[0]
	r.Port = new(Port)
	if err := r.Port.UnmarshalBinary(payload[8:]); err != nil {
		return err
	}

	return nil
}
