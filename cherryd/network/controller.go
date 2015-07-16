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

package network

import (
	"github.com/superkkt/cherry/cherryd/log"
	"github.com/superkkt/cherry/cherryd/protocol"
	"golang.org/x/net/context"
	"net"
)

type EventListener interface {
	ControllerEventListener
	TopologyEventListener
}

type ControllerEventListener interface {
	OnPacketIn(Finder, *Port, *protocol.Ethernet) error
	OnPortUp(Finder, *Port) error
	OnPortDown(Finder, *Port) error
	OnDeviceUp(Finder, *Device) error
	OnDeviceDown(Finder, *Device) error
}

type TopologyEventListener interface {
	OnTopologyChange(Finder) error
}

type Controller struct {
	log      log.Logger
	topo     *topology
	listener EventListener
}

func NewController(log log.Logger) *Controller {
	if log == nil {
		panic("Logger is nil")
	}

	return &Controller{
		log:  log,
		topo: newTopology(log),
	}
}

func (r *Controller) AddConnection(ctx context.Context, c net.Conn) {
	conf := sessionConfig{
		conn:     c,
		logger:   r.log,
		watcher:  r.topo,
		finder:   r.topo,
		listener: r.listener,
	}
	session := newSession(conf)
	go session.Run(ctx)
}

func (r *Controller) SetEventListener(l EventListener) {
	r.listener = l
	r.topo.setEventListener(l)
}

func (r *Controller) String() string {
	return r.topo.String()
}
