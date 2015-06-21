/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service, Inc. All rights reserved. 
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
