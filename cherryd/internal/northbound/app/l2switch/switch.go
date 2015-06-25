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

package l2switch

import (
	"bytes"
	"fmt"
	"github.com/dlintw/goconf"
	"github.com/superkkt/cherry/cherryd/internal/log"
	"github.com/superkkt/cherry/cherryd/internal/network"
	"github.com/superkkt/cherry/cherryd/internal/northbound/app"
	"github.com/superkkt/cherry/cherryd/openflow"
	"github.com/superkkt/cherry/cherryd/protocol"
	"net"
)

type L2Switch struct {
	app.BaseProcessor
	conf *goconf.ConfigFile
	log  log.Logger
}

func New(conf *goconf.ConfigFile, log log.Logger) *L2Switch {
	return &L2Switch{
		conf: conf,
		log:  log,
	}
}

func (r *L2Switch) Init() error {
	return nil
}

func (r *L2Switch) Name() string {
	return "L2Switch"
}

func flood(ingress *network.Port, packet []byte) error {
	f := ingress.Device().Factory()

	inPort := openflow.NewInPort()
	inPort.SetValue(ingress.Number())

	outPort := openflow.NewOutPort()
	outPort.SetFlood()

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

	return ingress.Device().SendMessage(out)
}

func packetout(egress *network.Port, packet []byte) error {
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

func isBroadcast(eth *protocol.Ethernet) bool {
	return bytes.Compare(eth.DstMAC, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}) == 0
}

type flowParam struct {
	device    *network.Device
	etherType uint16
	inPort    uint32
	outPort   uint32
	srcMAC    net.HardwareAddr
	dstMAC    net.HardwareAddr
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

	outPort := openflow.NewOutPort()
	outPort.SetValue(p.outPort)
	action, err := f.NewAction()
	if err != nil {
		return err
	}
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
	flow.SetPriority(10)
	flow.SetFlowMatch(match)
	flow.SetFlowInstruction(inst)

	return p.device.SendMessage(flow)
}

func setFlowRule(p flowParam) error {
	// Forward
	if err := installFlow(p); err != nil {
		return err
	}
	// Backward
	return installFlow(p)
}

type switchParam struct {
	finder    network.Finder
	ethernet  *protocol.Ethernet
	ingress   *network.Port
	egress    *network.Port
	rawPacket []byte
}

func (r *L2Switch) switching(p switchParam) error {
	// Find path between the ingress device and the other one that has that destination node
	path := p.finder.Path(p.ingress.Device().ID(), p.egress.Device().ID())
	if path == nil || len(path) == 0 {
		r.log.Debug(fmt.Sprintf("Not found a path from %v to %v", p.ethernet.SrcMAC, p.ethernet.DstMAC))
		return nil
	}
	// Drop this packet if it goes back to the ingress port to avoid duplicated packet routing
	if p.ingress.Number() == path[0][0].Number() {
		r.log.Debug("Ignore routing path that goes back to the ingress port")
		return nil
	}

	inPort := p.ingress.Number()
	// Install bi-directional flow rules into all devices on the path
	for _, v := range path {
		param := flowParam{
			device:    v[0].Device(),
			etherType: p.ethernet.Type,
			inPort:    inPort,
			outPort:   v[0].Number(),
			srcMAC:    p.ethernet.SrcMAC,
			dstMAC:    p.ethernet.DstMAC,
		}
		if err := setFlowRule(param); err != nil {
			return err
		}
		inPort = v[1].Number()
	}

	// Set final flow rule on the destination device
	param := flowParam{
		device:    p.egress.Device(),
		etherType: p.ethernet.Type,
		inPort:    inPort,
		outPort:   p.egress.Number(),
		srcMAC:    p.ethernet.SrcMAC,
		dstMAC:    p.ethernet.DstMAC,
	}
	if err := setFlowRule(param); err != nil {
		return err
	}

	// Send this ethernet packet directly to the destination node
	return packetout(p.egress, p.rawPacket)
}

func (r *L2Switch) localSwitching(p switchParam) error {
	param := flowParam{
		device:    p.ingress.Device(),
		etherType: p.ethernet.Type,
		inPort:    p.ingress.Number(),
		outPort:   p.egress.Number(),
		srcMAC:    p.ethernet.SrcMAC,
		dstMAC:    p.ethernet.DstMAC,
	}
	if err := setFlowRule(param); err != nil {
		return err
	}

	// Send this ethernet packet directly to the destination node
	return packetout(p.egress, p.rawPacket)
}

func (r *L2Switch) OnPacketIn(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet) error {
	drop, err := r.processPacket(finder, ingress, eth)
	if drop || err != nil {
		return err
	}

	return r.BaseProcessor.OnPacketIn(finder, ingress, eth)
}

func (r *L2Switch) processPacket(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet) (drop bool, err error) {
	packet, err := eth.MarshalBinary()
	if err != nil {
		return false, err
	}

	dstNode := finder.Node(eth.DstMAC)
	// Unknown node or broadcast request?
	if dstNode == nil || isBroadcast(eth) {
		r.log.Debug(fmt.Sprintf("Broadcasting (dstMAC=%v)", eth.DstMAC))
		return true, flood(ingress, packet)
	}

	param := switchParam{
		finder:    finder,
		ethernet:  eth,
		ingress:   ingress,
		egress:    dstNode.Port(),
		rawPacket: packet,
	}
	// Two nodes on a same switch device?
	if ingress.Device().ID() == dstNode.Port().Device().ID() {
		err = r.localSwitching(param)
	} else {
		err = r.switching(param)
	}
	if err != nil {
		return false, fmt.Errorf("failed to switch a packet: %v", err)
	}

	return true, nil
}

func (r *L2Switch) OnPortDown(finder network.Finder, port *network.Port) error {
	// Remove flow rules related with this downed port
	if err := r.cleanup(finder, port); err != nil {
		return err
	}

	return r.BaseProcessor.OnPortDown(finder, port)
}

func (r *L2Switch) OnTopologyChange(finder network.Finder) error {
	// We should remove all edges from all switch devices when the network topology is changed.
	// Otherwise, installed flow rules in switches may result in incorrect packet routing based on the previous topology.
	if err := r.removeAllFlows(finder.Devices()); err != nil {
		return err
	}

	return r.BaseProcessor.OnTopologyChange(finder)
}

func (r *L2Switch) cleanup(finder network.Finder, port *network.Port) error {
	r.log.Debug(fmt.Sprintf("Cleaning up for %v..", port.ID()))

	nodes := port.Nodes()
	// Remove all flows related with the nodes that are connected to this port
	for _, n := range nodes {
		r.log.Debug(fmt.Sprintf("Removing all flows related with a node %v..", n.MAC()))

		if err := r.removeFlowRules(finder, n.MAC()); err != nil {
			r.log.Err(fmt.Sprintf("Failed to remove flows related with %v: %v", n.MAC(), err))
			continue
		}
	}

	return nil
}

func (r *L2Switch) removeAllFlows(devices []*network.Device) error {
	r.log.Debug("Removing all flows from all devices..")

	for _, d := range devices {
		if d.IsClosed() {
			continue
		}
		factory := d.Factory()
		// Wildcard match
		match, err := factory.NewMatch()
		if err != nil {
			return err
		}
		if err := r.removeFlow(d, match); err != nil {
			r.log.Err(fmt.Sprintf("Failed to remove flows on %v: %v", d.ID(), err))
			continue
		}
	}

	return nil
}

func (r *L2Switch) removeFlowRules(finder network.Finder, mac net.HardwareAddr) error {
	devices := finder.Devices()
	for _, d := range devices {
		r.log.Debug(fmt.Sprintf("Removing all flows related with a node %v on device %v..", mac, d.ID()))

		if d.IsClosed() {
			continue
		}
		factory := d.Factory()
		// Remove all flow rules whose source MAC address is mac in its flow match
		match, err := factory.NewMatch()
		if err != nil {
			return err
		}
		match.SetSrcMAC(mac)
		if err := r.removeFlow(d, match); err != nil {
			return err
		}

		// Remove all flow rules whose destination MAC address is mac in its flow match
		match, err = factory.NewMatch()
		if err != nil {
			return err
		}
		match.SetDstMAC(mac)
		if err := r.removeFlow(d, match); err != nil {
			return err
		}
	}

	return nil
}

func (r *L2Switch) removeFlow(d *network.Device, match openflow.Match) error {
	r.log.Debug(fmt.Sprintf("Removing flows on device %v..", d.ID()))

	f := d.Factory()
	flowmod, err := f.NewFlowMod(openflow.FlowDelete)
	if err != nil {
		return err
	}
	// Remove flows except the table miss flows (Note that MSB of the cookie is a marker)
	flowmod.SetCookieMask(0x1 << 63)
	flowmod.SetTableID(0xFF) // ALL
	flowmod.SetFlowMatch(match)

	return d.SendMessage(flowmod)
}
