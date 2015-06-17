/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding"
)

type FlowModCmd uint8

const (
	FlowAdd FlowModCmd = iota
	FlowModify
	FlowDelete
)

type FlowMod interface {
	Cookie() uint64
	CookieMask() uint64
	encoding.BinaryMarshaler
	Error() error
	FlowInstruction() Instruction
	FlowMatch() Match
	HardTimeout() uint16
	Header
	IdleTimeout() uint16
	OutPort() OutPort
	Priority() uint16
	SetCookie(cookie uint64)
	SetCookieMask(mask uint64)
	SetFlowInstruction(action Instruction)
	SetFlowMatch(match Match)
	SetHardTimeout(timeout uint16)
	SetIdleTimeout(timeout uint16)
	SetOutPort(port OutPort)
	SetPriority(priority uint16)
	SetTableID(id uint8)
	TableID() uint8
}
