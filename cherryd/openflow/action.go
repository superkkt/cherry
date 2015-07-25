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
	"fmt"
	"net"
)

type Action interface {
	DstMAC() (ok bool, mac net.HardwareAddr)
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	Queue() (ok bool, queue uint32)
	// Error() returns last error message
	Error() error
	OutPort() OutPort
	SetDstMAC(mac net.HardwareAddr)
	SetQueue(queue uint32)
	SetOutPort(port OutPort)
	SetSrcMAC(mac net.HardwareAddr)
	SetVLANID(vid uint16)
	SrcMAC() (ok bool, mac net.HardwareAddr)
	VLANID() (ok bool, vid uint16)
}

type BaseAction struct {
	err    error
	output OutPort
	srcMAC *net.HardwareAddr
	dstMAC *net.HardwareAddr
	queue  int64
	vlanID int32
}

func NewBaseAction() *BaseAction {
	return &BaseAction{
		queue:  -1,
		vlanID: -1,
	}
}

func (r *BaseAction) VLANID() (ok bool, vid uint16) {
	if r.vlanID == -1 {
		return false, 0
	}

	return true, uint16(r.vlanID)
}

func (r *BaseAction) SetVLANID(vid uint16) {
	r.vlanID = int32(vid)
}

func (r *BaseAction) Queue() (ok bool, queue uint32) {
	if r.queue == -1 {
		return false, 0
	}

	return true, uint32(r.queue)
}

func (r *BaseAction) SetQueue(queue uint32) {
	r.queue = int64(queue)
}

func (r *BaseAction) SetOutPort(port OutPort) {
	r.output = port
}

func (r *BaseAction) OutPort() OutPort {
	return r.output
}

func (r *BaseAction) SetSrcMAC(mac net.HardwareAddr) {
	if mac == nil || len(mac) < 6 {
		r.err = fmt.Errorf("SetSrcMAC: %v", ErrInvalidMACAddress)
		return
	}

	r.srcMAC = &mac
}

func (r *BaseAction) SrcMAC() (ok bool, mac net.HardwareAddr) {
	if r.srcMAC == nil {
		return false, net.HardwareAddr([]byte{0, 0, 0, 0, 0, 0})
	}

	return true, *r.srcMAC
}

func (r *BaseAction) SetDstMAC(mac net.HardwareAddr) {
	if mac == nil || len(mac) < 6 {
		r.err = fmt.Errorf("SetDstMAC: %v", ErrInvalidMACAddress)
		return
	}

	r.dstMAC = &mac
}

func (r *BaseAction) DstMAC() (ok bool, mac net.HardwareAddr) {
	if r.dstMAC == nil {
		return false, net.HardwareAddr([]byte{0, 0, 0, 0, 0, 0})
	}

	return true, *r.dstMAC
}

func (r *BaseAction) Error() error {
	return r.err
}
