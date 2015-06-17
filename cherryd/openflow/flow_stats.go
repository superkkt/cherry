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
	Cookie() uint64
	CookieMask() uint64
	encoding.BinaryMarshaler
	Error() error
	Header
	Match() Match
	SetCookie(cookie uint64)
	SetCookieMask(mask uint64)
	SetMatch(match Match)
	// 0xFF means all table
	SetTableID(id uint8)
	TableID() uint8
}

// TODO: Implement FlowStatsReply
