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

package app

import (
	"github.com/superkkt/cherry/cherryd/network"
	"github.com/superkkt/cherry/cherryd/openflow"
	"github.com/superkkt/cherry/cherryd/protocol"
)

// Processor should prepare to be executed by multiple goroutines simultaneously.
type Processor interface {
	network.EventListener
	Init() error
	// Name returns the application name that is globally unique
	Name() string
	Next() (next Processor, ok bool)
	SetNext(Processor)
}

type BaseProcessor struct {
	next Processor
}

func (r *BaseProcessor) Init() error {
	return nil
}

func (r *BaseProcessor) Name() string {
	return "BaseProcessor"
}

func (r *BaseProcessor) OnPacketIn(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet) error {
	// Do nothging and execute the next processor if it exists
	next, ok := r.Next()
	if !ok {
		return nil
	}
	return next.OnPacketIn(finder, ingress, eth)
}

func (r *BaseProcessor) OnDeviceUp(finder network.Finder, device *network.Device) error {
	// Do nothging and execute the next processor if it exists
	next, ok := r.Next()
	if !ok {
		return nil
	}
	return next.OnDeviceUp(finder, device)
}

func (r *BaseProcessor) OnDeviceDown(finder network.Finder, device *network.Device) error {
	// Do nothging and execute the next processor if it exists
	next, ok := r.Next()
	if !ok {
		return nil
	}
	return next.OnDeviceDown(finder, device)
}

func (r *BaseProcessor) OnPortUp(finder network.Finder, port *network.Port) error {
	// Do nothging and execute the next processor if it exists
	next, ok := r.Next()
	if !ok {
		return nil
	}
	return next.OnPortUp(finder, port)
}

func (r *BaseProcessor) OnPortDown(finder network.Finder, port *network.Port) error {
	// Do nothging and execute the next processor if it exists
	next, ok := r.Next()
	if !ok {
		return nil
	}
	return next.OnPortDown(finder, port)
}

func (r *BaseProcessor) OnTopologyChange(finder network.Finder) error {
	// Do nothging and execute the next processor if it exists
	next, ok := r.Next()
	if !ok {
		return nil
	}
	return next.OnTopologyChange(finder)
}

func (r *BaseProcessor) Next() (next Processor, ok bool) {
	if r.next != nil {
		return r.next, true
	}

	return nil, false
}

func (r *BaseProcessor) SetNext(next Processor) {
	r.next = next
}

func (r *BaseProcessor) PacketOut(egress *network.Port, packet []byte) error {
	f := egress.Device().Factory()

	inPort := openflow.NewInPort()
	inPort.SetController()

	outPort := openflow.NewOutPort()
	outPort.SetValue(egress.Number())

	action, err := f.NewAction()
	if err != nil {
		return err
	}
	action.SetOutPort(outPort)

	out, err := f.NewPacketOut()
	if err != nil {
		return err
	}
	out.SetInPort(inPort)
	out.SetAction(action)
	out.SetData(packet)

	return egress.Device().SendMessage(out)
}
