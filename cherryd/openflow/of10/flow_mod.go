/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of10

import (
	"encoding/binary"
	"errors"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type FlowModConfig struct {
	Cookie      uint64
	IdleTimeout uint16
	HardTimeout uint16
	Priority    uint16
	Match       openflow.Match
	Action      openflow.Action
}

type FlowMod struct {
	openflow.Message
	command uint16
	config  *FlowModConfig
}

func newFlowMod(xid uint32, command uint16, config *FlowModConfig) *FlowMod {
	return &FlowMod{
		Message: openflow.NewMessage(openflow.Ver10, OFPT_FLOW_MOD, xid),
		command: command,
		config:  config,
	}
}

func NewFlowModAdd(xid uint32, config *FlowModConfig) *FlowMod {
	return newFlowMod(xid, OFPFC_ADD, config)
}

func NewFlowModModify(xid uint32, config *FlowModConfig) *FlowMod {
	return newFlowMod(xid, OFPFC_MODIFY, config)
}

func NewFlowModDelete(xid uint32, config *FlowModConfig) *FlowMod {
	return newFlowMod(xid, OFPFC_DELETE, config)
}

func (r *FlowMod) MarshalBinary() ([]byte, error) {
	if r.config.Match == nil {
		return nil, errors.New("empty flow match")
	}

	v, err := r.config.Match.MarshalBinary()
	if err != nil {
		return nil, err
	}

	v = append(v, make([]byte, 24)...)
	binary.BigEndian.PutUint64(v[0:8], r.config.Cookie)
	binary.BigEndian.PutUint16(v[8:10], r.command)
	binary.BigEndian.PutUint16(v[10:12], r.config.IdleTimeout)
	binary.BigEndian.PutUint16(v[12:14], r.config.HardTimeout)
	binary.BigEndian.PutUint16(v[14:16], r.config.Priority)
	binary.BigEndian.PutUint32(v[16:20], OFP_NO_BUFFER)
	binary.BigEndian.PutUint16(v[20:22], OFPP_NONE)
	binary.BigEndian.PutUint16(v[22:24], OFPFF_SEND_FLOW_REM|OFPFF_CHECK_OVERLAP)

	if r.config.Action != nil {
		action, err := r.config.Action.MarshalBinary()
		if err != nil {
			return nil, err
		}
		v = append(v, action...)
	}

	r.SetPayload(v)
	return r.Message.MarshalBinary()
}
