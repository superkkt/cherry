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

type Instruction interface {
	encoding.BinaryMarshaler
}

type FlowMod interface {
	Header
	Cookie() uint64
	SetCookie(cookie uint64) error
	CookieMask() uint64
	SetCookieMask(mask uint64) error
	TableID() uint8
	SetTableID(id uint8) error
	IdleTimeout() uint16
	SetIdleTimeout(timeout uint16) error
	HardTimeout() uint16
	SetHardTimeout(timeout uint16) error
	Priority() uint16
	SetPriority(priority uint16) error
	FlowMatch() Match
	SetFlowMatch(match Match) error
	FlowAction() Instruction
	SetFlowAction(action Instruction) error
	encoding.BinaryMarshaler
}
