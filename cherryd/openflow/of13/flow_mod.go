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
	err error
	openflow.Message
	command     uint8
	cookie      uint64
	cookieMask  uint64
	tableID     uint8
	idleTimeout uint16
	hardTimeout uint16
	priority    uint16
	match       openflow.Match
	instruction openflow.Instruction
	outPort     openflow.OutPort
}

func NewFlowMod(xid uint32, cmd uint8) openflow.FlowMod {
	// Default out_port value is OFPP_NONE (OFPP_ANY)
	outPort := openflow.NewOutPort()
	outPort.SetNone()

	return &FlowMod{
		Message: openflow.NewMessage(openflow.OF13_VERSION, OFPT_FLOW_MOD, xid),
		command: cmd,
		outPort: outPort,
	}
}

func (r *FlowMod) Error() error {
	return r.err
}

func (r *FlowMod) Cookie() uint64 {
	return r.cookie
}

func (r *FlowMod) SetCookie(cookie uint64) {
	r.cookie = cookie
}

func (r *FlowMod) CookieMask() uint64 {
	return r.cookieMask
}

func (r *FlowMod) SetCookieMask(mask uint64) {
	r.cookieMask = mask
}

func (r *FlowMod) TableID() uint8 {
	return r.tableID
}

func (r *FlowMod) SetTableID(id uint8) {
	r.tableID = id
}

func (r *FlowMod) IdleTimeout() uint16 {
	return r.idleTimeout
}

func (r *FlowMod) SetIdleTimeout(timeout uint16) {
	r.idleTimeout = timeout
}

func (r *FlowMod) HardTimeout() uint16 {
	return r.hardTimeout
}

func (r *FlowMod) SetHardTimeout(timeout uint16) {
	r.hardTimeout = timeout
}

func (r *FlowMod) Priority() uint16 {
	return r.priority
}

func (r *FlowMod) SetPriority(priority uint16) {
	r.priority = priority
}

func (r *FlowMod) FlowMatch() openflow.Match {
	return r.match
}

func (r *FlowMod) SetFlowMatch(match openflow.Match) {
	if match == nil {
		panic("flow match is nil")
	}
	r.match = match
}

func (r *FlowMod) FlowInstruction() openflow.Instruction {
	return r.instruction
}

func (r *FlowMod) SetFlowInstruction(inst openflow.Instruction) {
	if inst == nil {
		panic("flow instruction is nil")
	}
	r.instruction = inst
}

func (r *FlowMod) OutPort() openflow.OutPort {
	return r.outPort
}

func (r *FlowMod) SetOutPort(p openflow.OutPort) {
	r.outPort = p
}

func (r *FlowMod) MarshalBinary() ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}

	v := make([]byte, 40)
	binary.BigEndian.PutUint64(v[0:8], r.cookie)
	binary.BigEndian.PutUint64(v[8:16], r.cookieMask)
	v[16] = r.tableID
	v[17] = r.command
	binary.BigEndian.PutUint16(v[18:20], r.idleTimeout)
	binary.BigEndian.PutUint16(v[20:22], r.hardTimeout)
	binary.BigEndian.PutUint16(v[22:24], r.priority)
	binary.BigEndian.PutUint32(v[24:28], OFP_NO_BUFFER)
	if r.outPort.IsNone() {
		binary.BigEndian.PutUint32(v[28:32], OFPP_ANY)
	} else {
		binary.BigEndian.PutUint32(v[28:32], r.outPort.Value())
	}
	binary.BigEndian.PutUint32(v[32:36], OFPP_ANY)
	// XXX: EdgeCore AS4600-54T switch does not support OFPFF_CHECK_OVERLAP
	binary.BigEndian.PutUint16(v[36:38], OFPFF_SEND_FLOW_REM)
	// v[38:40] is padding

	if r.match == nil {
		return nil, errors.New("empty flow match")
	}
	match, err := r.match.MarshalBinary()
	if err != nil {
		return nil, err
	}
	v = append(v, match...)
	if r.instruction != nil {
		ins, err := r.instruction.MarshalBinary()
		if err != nil {
			return nil, err
		}
		v = append(v, ins...)
	}

	r.SetPayload(v)
	return r.Message.MarshalBinary()
}
