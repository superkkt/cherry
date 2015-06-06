/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

import (
	"encoding/binary"
	"errors"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type FlowMod struct {
	openflow.Message
	command     uint8
	cookie      uint64
	cookieMask  uint64
	tableID     uint8
	idleTimeout uint16
	hardTimeout uint16
	priority    uint16
	match       openflow.Match
	action      openflow.Instruction
}

func NewFlowMod(xid uint32, cmd uint8) openflow.FlowMod {
	return &FlowMod{
		Message: openflow.NewMessage(openflow.OF13_VERSION, OFPT_FLOW_MOD, xid),
		command: cmd,
	}
}

func (r FlowMod) Cookie() uint64 {
	return r.cookie
}

func (r *FlowMod) SetCookie(cookie uint64) error {
	r.cookie = cookie
	return nil
}

func (r FlowMod) CookieMask() uint64 {
	return r.cookieMask
}

func (r *FlowMod) SetCookieMask(mask uint64) error {
	r.cookieMask = mask
	return nil
}

func (r FlowMod) TableID() uint8 {
	return r.tableID
}

func (r *FlowMod) SetTableID(id uint8) error {
	r.tableID = id
	return nil
}

func (r FlowMod) IdleTimeout() uint16 {
	return r.idleTimeout
}

func (r *FlowMod) SetIdleTimeout(timeout uint16) error {
	r.idleTimeout = timeout
	return nil
}

func (r FlowMod) HardTimeout() uint16 {
	return r.hardTimeout
}

func (r *FlowMod) SetHardTimeout(timeout uint16) error {
	r.hardTimeout = timeout
	return nil
}

func (r FlowMod) Priority() uint16 {
	return r.priority
}

func (r *FlowMod) SetPriority(priority uint16) error {
	r.priority = priority
	return nil
}

func (r FlowMod) FlowMatch() openflow.Match {
	return r.match
}

func (r *FlowMod) SetFlowMatch(match openflow.Match) error {
	if match == nil {
		return errors.New("flow match is nil")
	}
	r.match = match
	return nil
}

func (r FlowMod) FlowAction() openflow.Instruction {
	return r.action
}

func (r *FlowMod) SetFlowAction(action openflow.Instruction) error {
	r.action = action
	return nil
}

func (r *FlowMod) MarshalBinary() ([]byte, error) {
	v := make([]byte, 40)
	binary.BigEndian.PutUint64(v[0:8], r.cookie)
	binary.BigEndian.PutUint64(v[8:16], r.cookieMask)
	v[16] = r.tableID
	v[17] = r.command
	binary.BigEndian.PutUint16(v[18:20], r.idleTimeout)
	binary.BigEndian.PutUint16(v[20:22], r.hardTimeout)
	binary.BigEndian.PutUint16(v[22:24], r.priority)
	binary.BigEndian.PutUint32(v[24:28], OFP_NO_BUFFER)
	binary.BigEndian.PutUint32(v[28:32], OFPP_ANY)
	binary.BigEndian.PutUint32(v[32:36], OFPP_ANY)
	// XXX: EdgeCore AS4600-54T switch does not support OFPFF_CHECK_OVERLAP
	binary.BigEndian.PutUint16(v[36:38], OFPFF_SEND_FLOW_REM)
	// v[38:40] is padding
	match, err := r.match.MarshalBinary()
	if err != nil {
		return nil, err
	}
	v = append(v, match...)
	if r.action != nil {
		ins, err := r.action.MarshalBinary()
		if err != nil {
			return nil, err
		}
		v = append(v, ins...)
	}

	r.SetPayload(v)
	return r.Message.MarshalBinary()
}
