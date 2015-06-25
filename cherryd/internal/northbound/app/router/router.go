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

package router

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/dlintw/goconf"
	"github.com/superkkt/cherry/cherryd/internal/log"
	"github.com/superkkt/cherry/cherryd/internal/network"
	"github.com/superkkt/cherry/cherryd/internal/northbound/app"
	"github.com/superkkt/cherry/cherryd/openflow"
	"github.com/superkkt/cherry/cherryd/protocol"
	"net"
)

// TODO: Implement Border Gateway Protocol (BGP) to directly communicate with external routers
type Router struct {
	app.BaseProcessor
	conf *goconf.ConfigFile
	log  log.Logger
	db   database
	// Virtual MAC address
	mac net.HardwareAddr
}

type database interface {
	FindMAC(ip net.IP) (mac net.HardwareAddr, ok bool, err error)
	GetNetworks() ([]*net.IPNet, error)
}

func New(conf *goconf.ConfigFile, log log.Logger, db database) *Router {
	return &Router{
		conf: conf,
		log:  log,
		db:   db,
	}
}

func (r *Router) Name() string {
	return "Router"
}

func (r *Router) Init() error {
	mac, err := r.conf.GetString("router", "mac")
	if err != nil || len(mac) == 0 {
		return errors.New("empty virtual MAC address of the router in the config file")
	}
	r.mac, err = net.ParseMAC(mac)
	if err != nil {
		return err
	}

	return nil
}

func (r *Router) OnPacketIn(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet) error {
	// Is this packet going to the router?
	if bytes.Compare(eth.DstMAC, r.mac) != 0 {
		return r.BaseProcessor.OnPacketIn(finder, ingress, eth)
	}

	// IPv4?
	if eth.Type != 0x0800 {
		// Drop the packet if it is not an IPv4 packet
		return nil
	}
	ipv4 := new(protocol.IPv4)
	if err := ipv4.UnmarshalBinary(eth.Payload); err != nil {
		return err
	}

	networks, err := r.db.GetNetworks()
	if err != nil {
		return err
	}
	if isMyNetwork(networks, ipv4.DstIP) {
		return r.handleIncoming(finder, ingress, eth, ipv4)
	} else {
		return r.handleOutgoing(finder, ingress, eth, ipv4)
	}
}

func isMyNetwork(networks []*net.IPNet, ip net.IP) bool {
	for _, n := range networks {
		if n.Contains(ip) {
			return true
		}
	}

	return false
}

func (r *Router) handleIncoming(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet, ipv4 *protocol.IPv4) error {
	mac, ok, err := r.db.FindMAC(ipv4.DstIP)
	if err != nil {
		return err
	}
	if !ok {
		r.log.Debug(fmt.Sprintf("Router: incoming packet to an unknown host %v from %v", ipv4.DstIP, ipv4.SrcIP))
		return nil
	}
	r.log.Debug(fmt.Sprintf("Router: forwarding to a host.. IP=%v, MAC=%v", ipv4.DstIP, mac))

	// Do we have the destination node?
	dstNode := finder.Node(mac)
	if dstNode == nil {
		r.log.Debug(fmt.Sprintf("Router: we don't know where the host is connected.. IP=%v, MAC=%v", ipv4.DstIP, mac))
		return r.BaseProcessor.OnPacketIn(finder, ingress, eth)
	}
	// Two nodes on a same switch device?
	if ingress.Device().ID() == dstNode.Port().Device().ID() {
		param := flowParam{
			device:    ingress.Device(),
			etherType: eth.Type,
			inPort:    ingress.Number(),
			outPort:   dstNode.Port().Number(),
			srcMAC:    eth.SrcMAC,
			dstMAC:    eth.DstMAC,
			newDstMAC: mac,
			dstIP:     &net.IPNet{IP: ipv4.DstIP, Mask: net.IPv4Mask(255, 255, 255, 255)},
		}
		if err := installFlow(param); err != nil {
			return err
		}
		// TODO: packet out
	}

	path := finder.Path(ingress.Device().ID(), dstNode.Port().Device().ID())
	if path == nil || len(path) == 0 {
		r.log.Info(fmt.Sprintf("Router: not found a path from %v to %v", ingress.ID(), dstNode.Port().ID()))
		return nil
	}

	// TODO: install a flow rule

	// TODO: set ingress to next device's port and pass the ethernet packet whose dst MAC is modified to the next app

	return nil
}

func (r *Router) handleOutgoing(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet, ipv4 *protocol.IPv4) error {
	return nil
}

type flowParam struct {
	device    *network.Device
	etherType uint16
	inPort    uint32
	outPort   uint32
	srcMAC    net.HardwareAddr
	dstMAC    net.HardwareAddr
	newDstMAC net.HardwareAddr
	dstIP     *net.IPNet
}

func installFlow(p flowParam) error {
	f := p.device.Factory()

	inPort := openflow.NewInPort()
	inPort.SetValue(p.inPort)
	match, err := f.NewMatch()
	if err != nil {
		return err
	}
	match.SetInPort(inPort)
	match.SetEtherType(p.etherType)
	match.SetSrcMAC(p.srcMAC)
	match.SetDstMAC(p.dstMAC)
	match.SetDstIP(p.dstIP)

	outPort := openflow.NewOutPort()
	outPort.SetValue(p.outPort)
	action, err := f.NewAction()
	if err != nil {
		return err
	}
	action.SetDstMAC(p.newDstMAC)
	action.SetOutPort(outPort)
	inst, err := f.NewInstruction()
	if err != nil {
		return err
	}
	inst.ApplyAction(action)

	flow, err := f.NewFlowMod(openflow.FlowAdd)
	if err != nil {
		return err
	}
	flow.SetTableID(p.device.FlowTableID())
	flow.SetIdleTimeout(30)
	// This priority should be higher than L2 switch module's one
	flow.SetPriority(30)
	flow.SetFlowMatch(match)
	flow.SetFlowInstruction(inst)

	return p.device.SendMessage(flow)
}
