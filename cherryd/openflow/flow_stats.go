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

type FlowStatsRequest interface {
	Header
	Cookie() uint64
	SetCookie(cookie uint64) error
	CookieMask() uint64
	SetCookieMask(mask uint64) error
	Match() Match
	SetMatch(match Match) error
	TableID() uint8
	// 0xFF means all table
	SetTableID(id uint8) error
	encoding.BinaryMarshaler
}

// TODO: Implement FlowStatsReply
