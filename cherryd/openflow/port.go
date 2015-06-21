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
	"net"
)

const (
	table = iota
	flood
	all
	controller
	none
)

type OutPort struct {
	logical uint8
	value   uint32
}

// NewOutPort returns output port whose default value is FLOOD
func NewOutPort() OutPort {
	return OutPort{
		logical: 0x1 << flood,
	}
}

func (r *OutPort) SetTable() {
	r.logical = 0x1 << table
}

func (r *OutPort) IsTable() bool {
	return r.logical&(0x1<<table) != 0
}

func (r *OutPort) SetFlood() {
	r.logical = 0x1 << flood
}

func (r *OutPort) IsFlood() bool {
	return r.logical&(0x1<<flood) != 0
}

func (r *OutPort) SetAll() {
	r.logical = 0x1 << all
}

func (r *OutPort) IsAll() bool {
	return r.logical&(0x1<<all) != 0
}

func (r *OutPort) SetController() {
	r.logical = 0x1 << controller
}

func (r *OutPort) IsController() bool {
	return r.logical&(0x1<<controller) != 0
}

func (r *OutPort) SetNone() {
	r.logical = 0x1 << none
}

func (r *OutPort) IsNone() bool {
	return r.logical&(0x1<<none) != 0
}

func (r *OutPort) SetValue(port uint32) {
	r.logical = 0x0
	r.value = port
}

func (r *OutPort) Value() uint32 {
	return r.value
}

type InPort struct {
	value      uint32
	controller bool
}

func NewInPort() InPort {
	return InPort{
		controller: true,
	}
}

func (r *InPort) SetValue(port uint32) {
	r.controller = false
	r.value = port
}

func (r *InPort) SetController() {
	r.controller = true
	r.value = 0
}

func (r *InPort) IsController() bool {
	return r.controller
}

func (r *InPort) Value() uint32 {
	return r.value
}

type Port interface {
	Number() uint32
	MAC() net.HardwareAddr
	Name() string
	IsPortDown() bool // Is the port Administratively down?
	IsLinkDown() bool // Is a physical link on the port down?
	IsCopper() bool
	IsFiber() bool
	IsAutoNego() bool
	// Speed returns current link speed in MB
	Speed() uint64
	encoding.BinaryUnmarshaler
}
