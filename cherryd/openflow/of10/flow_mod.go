/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service 
 * Kitae Kim <superkkt@sds.co.kr>
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along
 * with this program; if not, write to the Free Software Foundation, Inc.,
 * 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
 */

package of10

import (
	"encoding/binary"
	"errors"
	"github.com/superkkt/cherry/cherryd/openflow"
)

type FlowMod struct {
	err error
	openflow.Message
	command     uint16
	cookie      uint64
	idleTimeout uint16
	hardTimeout uint16
	priority    uint16
	match       openflow.Match
	instruction openflow.Instruction
	outPort     openflow.OutPort
}

func NewFlowMod(xid uint32, cmd uint16) openflow.FlowMod {
	// Default out_port value is OFPP_NONE (OFPP_ANY)
	outPort := openflow.NewOutPort()
	outPort.SetNone()

	return &FlowMod{
		Message: openflow.NewMessage(openflow.OF10_VERSION, OFPT_FLOW_MOD, xid),
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
	// OpenFlow 1.0 does not have the cookie mask
	return 0
}

func (r *FlowMod) SetCookieMask(mask uint64) {
	// OpenFlow 1.0 does not have the cookie mask
}

func (r *FlowMod) TableID() uint8 {
	// OpenFlow 1.0 does not have table ID
	return 0
}

func (r *FlowMod) SetTableID(id uint8) {
	// OpenFlow 1.0 does not have table ID
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

	v := make([]byte, 24)
	binary.BigEndian.PutUint64(v[0:8], r.cookie)
	binary.BigEndian.PutUint16(v[8:10], r.command)
	binary.BigEndian.PutUint16(v[10:12], r.idleTimeout)
	binary.BigEndian.PutUint16(v[12:14], r.hardTimeout)
	binary.BigEndian.PutUint16(v[14:16], r.priority)
	binary.BigEndian.PutUint32(v[16:20], OFP_NO_BUFFER)
	if r.outPort.IsNone() {
		binary.BigEndian.PutUint16(v[20:22], OFPP_NONE)
	} else {
		binary.BigEndian.PutUint16(v[20:22], uint16(r.outPort.Value()))
	}
	binary.BigEndian.PutUint16(v[22:24], OFPFF_SEND_FLOW_REM)

	if r.match == nil {
		return nil, errors.New("empty flow match")
	}
	result, err := r.match.MarshalBinary()
	if err != nil {
		return nil, err
	}
	result = append(result, v...)

	if r.instruction != nil {
		instruction, err := r.instruction.MarshalBinary()
		if err != nil {
			return nil, err
		}
		result = append(result, instruction...)
	}

	r.SetPayload(result)
	return r.Message.MarshalBinary()
}
