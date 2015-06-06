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

type FlowMod struct {
	openflow.Message
	command     uint16
	cookie      uint64
	idleTimeout uint16
	hardTimeout uint16
	priority    uint16
	match       openflow.Match
	action      openflow.Instruction
}

func NewFlowMod(xid uint32, cmd uint16) openflow.FlowMod {
	return &FlowMod{
		Message: openflow.NewMessage(openflow.OF10_VERSION, OFPT_FLOW_MOD, xid),
		command: cmd,
	}
}

func (r *FlowMod) Cookie() uint64 {
	return r.cookie
}

func (r *FlowMod) SetCookie(cookie uint64) error {
	r.cookie = cookie
	return nil
}

func (r *FlowMod) CookieMask() uint64 {
	// OpenFlow 1.0 does not have the cookie mask
	return 0
}

func (r *FlowMod) SetCookieMask(mask uint64) error {
	// OpenFlow 1.0 does not have the cookie mask
	return nil
}

func (r *FlowMod) TableID() uint8 {
	// OpenFlow 1.0 does not have table ID
	return 0
}

func (r *FlowMod) SetTableID(id uint8) error {
	// OpenFlow 1.0 does not have table ID
	return nil
}

func (r *FlowMod) IdleTimeout() uint16 {
	return r.idleTimeout
}

func (r *FlowMod) SetIdleTimeout(timeout uint16) error {
	r.idleTimeout = timeout
	return nil
}

func (r *FlowMod) HardTimeout() uint16 {
	return r.hardTimeout
}

func (r *FlowMod) SetHardTimeout(timeout uint16) error {
	r.hardTimeout = timeout
	return nil
}

func (r *FlowMod) Priority() uint16 {
	return r.priority
}

func (r *FlowMod) SetPriority(priority uint16) error {
	r.priority = priority
	return nil
}

func (r *FlowMod) FlowMatch() openflow.Match {
	return r.match
}

func (r *FlowMod) SetFlowMatch(match openflow.Match) error {
	if match == nil {
		return errors.New("flow match is nil")
	}
	r.match = match
	return nil
}

func (r *FlowMod) FlowAction() openflow.Instruction {
	return r.action
}

func (r *FlowMod) SetFlowAction(action openflow.Instruction) error {
	r.action = action
	return nil
}

func (r *FlowMod) MarshalBinary() ([]byte, error) {
	v := make([]byte, 24)
	binary.BigEndian.PutUint64(v[0:8], r.cookie)
	binary.BigEndian.PutUint16(v[8:10], r.command)
	binary.BigEndian.PutUint16(v[10:12], r.idleTimeout)
	binary.BigEndian.PutUint16(v[12:14], r.hardTimeout)
	binary.BigEndian.PutUint16(v[14:16], r.priority)
	binary.BigEndian.PutUint32(v[16:20], OFP_NO_BUFFER)
	binary.BigEndian.PutUint16(v[20:22], OFPP_NONE)
	binary.BigEndian.PutUint16(v[22:24], OFPFF_SEND_FLOW_REM)
	result, err := r.match.MarshalBinary()
	if err != nil {
		return nil, err
	}
	result = append(result, v...)

	if r.action != nil {
		action, err := r.action.MarshalBinary()
		if err != nil {
			return nil, err
		}
		result = append(result, action...)
	}

	r.SetPayload(result)
	return r.Message.MarshalBinary()
}
