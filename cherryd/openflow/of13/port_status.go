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
	reason uint8
	port   openflow.Port
}

func (r PortStatus) Reason() openflow.PortReason {
	switch r.reason {
	case OFPPR_ADD:
		return openflow.PortAdded
	case OFPPR_DELETE:
		return openflow.PortDeleted
	case OFPPR_MODIFY:
		return openflow.PortModified
	default:
		return openflow.PortReason(r.reason)
	}
}

func (r PortStatus) Port() openflow.Port {
	return r.port
}

func (r *PortStatus) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 72 {
		return openflow.ErrInvalidPacketLength
	}
	r.reason = payload[0]
	r.port = new(Port)
	if err := r.port.UnmarshalBinary(payload[8:]); err != nil {
		return err
	}

	return nil
}
