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

type PropertyType uint16

const (
	OFPQT_NONE PropertyType = iota
	OFPQT_MIN_RATE
	OFPQT_MAX_RATE
	OFPQT_EXPERIMENTER = 0xffff
)

type Queue interface {
	ID() uint32
	Port() uint32
	Length() uint16
	Property() []QueueProperty
	encoding.BinaryUnmarshaler
}

type QueueProperty interface {
	Type() PropertyType
	Length() uint16
	Rate() (uint16, error)
	Experimenter() (uint32, error)
	Data() []byte
	encoding.BinaryUnmarshaler
}

type QueueGetConfigRequest interface {
	Header
	Port() OutPort
	SetPort(OutPort)
	encoding.BinaryMarshaler
}

type QueueGetConfigReply interface {
	Header
	Port() uint32
	Queue() []Queue
	encoding.BinaryUnmarshaler
}
