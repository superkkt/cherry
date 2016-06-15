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

type ConfigFlag uint16

const (
	FragNormal ConfigFlag = iota
	FragDrop
	FragReasm
	FragMask
)

type Config interface {
	// Error() returns last error message
	Error() error
	Flags() ConfigFlag
	MissSendLength() uint16
	SetFlags(flags ConfigFlag)
	SetMissSendLength(length uint16)
}

type SetConfig interface {
	Header
	Config
	encoding.BinaryMarshaler
}

type GetConfigRequest interface {
	Header
	encoding.BinaryMarshaler
}

type GetConfigReply interface {
	Header
	Config
	encoding.BinaryUnmarshaler
}
