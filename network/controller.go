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
	"context"
	"net"

	"github.com/superkkt/cherry/openflow"
	"github.com/superkkt/cherry/protocol"

	"github.com/superkkt/go-logging"
)

var (
	logger = logging.MustGetLogger("network")
)

type database interface {
	Location(mac net.HardwareAddr) (dpid string, port uint32, status LocationStatus, err error)
}

type LocationStatus int

const (
	// Unregistered MAC address.
	LocationUnregistered LocationStatus = iota
	// Registered MAC address, but we don't know its physical location yet.
	LocationUndiscovered
	// Registered MAC address, and we know its physical location.
	LocationDiscovered
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
	OnFlowRemoved(Finder, openflow.FlowRemoved) error
}

type TopologyEventListener interface {
	OnTopologyChange(Finder) error
}

type Controller struct {
	topo     *topology
	listener EventListener
}

func NewController(db database) *Controller {
	return &Controller{
		topo: newTopology(db),
	}
}

func (r *Controller) AddConnection(ctx context.Context, c net.Conn) {
	conf := sessionConfig{
		conn:     c,
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

func (r *Controller) Announce(ip net.IP, mac net.HardwareAddr) error {
	for _, device := range r.topo.Devices() {
		logger.Debugf("sending ARP announcement for a host (IP: %v, MAC: %v) via %v", ip, mac, device.ID())
		if err := device.SendARPAnnouncement(ip, mac); err != nil {
			logger.Errorf("failed to send ARP announcement via %v: %v", device.ID(), err)
			continue
		}
	}

	return nil
}

func (r *Controller) RemoveFlows() error {
	for _, device := range r.topo.Devices() {
		logger.Infof("removing all flows from %v", device.ID())
		if err := device.RemoveFlows(); err != nil {
			logger.Warningf("failed to remove all flows on %v device: %v", device.ID(), err)
			continue
		}
	}

	return nil
}

func (r *Controller) RemoveFlowsByMAC(mac net.HardwareAddr) error {
	for _, device := range r.topo.Devices() {
		if err := device.RemoveFlowByMAC(mac); err != nil {
			logger.Errorf("failed to remove flows for %v from %v: %v", mac, device.ID(), err)
			continue
		}
		logger.Debugf("removed flows whose destination MAC address is %v on %v", mac, device.ID())
	}

	return nil
}
