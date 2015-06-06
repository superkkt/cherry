/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding"
	"net"
)

type OutPort uint

const (
	OutToTable      OutPort = 0xfffffff9
	OutToAll                = 0xfffffffc
	OutToController         = 0xfffffffd
	OutToNone               = 0xffffffff
)

type InPort struct {
	port       uint32
	controller bool
}

func NewInPort() InPort {
	return InPort{
		controller: true,
	}
}

func (r *InPort) SetPort(port uint32) {
	r.controller = false
	r.port = port
}

func (r *InPort) IsController() bool {
	return r.controller
}

func (r *InPort) Port() uint32 {
	return r.port
}

type Port interface {
	Number() uint
	MAC() net.HardwareAddr
	Name() string
	IsPortDown() bool // Is the port Administratively down?
	IsLinkDown() bool // Is a physical link on the port down?
	IsCopper() bool
	IsFiber() bool
	IsAutoNego() bool
	Speed() uint64
	encoding.BinaryUnmarshaler
}
