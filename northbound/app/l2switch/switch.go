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
	"net"
	"sync"
	"time"

	"github.com/superkkt/cherry/network"
	"github.com/superkkt/cherry/northbound/app"
	"github.com/superkkt/cherry/openflow"
	"github.com/superkkt/cherry/protocol"

	"github.com/pkg/errors"
	"github.com/superkkt/go-logging"
)

var (
	logger = logging.MustGetLogger("l2switch")
)

type L2Switch struct {
	app.BaseProcessor
	stormCtrl *stormController
	db        Database
	once      sync.Once
}

type Database interface {
	// MACAddrs returns all the registered MAC addresses.
	MACAddrs() ([]net.HardwareAddr, error)
}

func New(db Database) *L2Switch {
	return &L2Switch{
		stormCtrl: newStormController(100, new(flooder)),
		db:        db,
	}
}

type flooder struct{}

// flood broadcasts packet to all ports on the ingress device, except the ingress port itself.
func (r *flooder) flood(ingress *network.Port, packet []byte) error {
	return ingress.Device().Flood(ingress, packet)
}

func (r *L2Switch) Init() error {
	return nil
}

func (r *L2Switch) Name() string {
	return "L2Switch"
}

func isBroadcast(eth *protocol.Ethernet) bool {
	return bytes.Compare(eth.DstMAC, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}) == 0
}

type flowParam struct {
	device  *network.Device
	dstMAC  net.HardwareAddr
	outPort uint32
}

func (r flowParam) String() string {
	return fmt.Sprintf("Device=%v, DstMAC=%v, OutPort=%v", r.device.ID(), r.dstMAC, r.outPort)
}

func (r *L2Switch) setFlow(p flowParam) error {
	f := p.device.Factory()
	match, err := f.NewMatch()
	if err != nil {
		return err
	}
	match.SetDstMAC(p.dstMAC)

	outPort := openflow.NewOutPort()
	outPort.SetValue(p.outPort)

	if err := p.device.SetFlow(match, outPort); err != nil {
		return err
	}
	logger.Debugf("installed a new flow rule: %v", p)

	return nil
}

type switchParam struct {
	finder    network.Finder
	ethernet  *protocol.Ethernet
	ingress   *network.Port
	egress    *network.Port
	rawPacket []byte
}

func (r *L2Switch) switching(p switchParam) error {
	param := flowParam{
		device:  p.ingress.Device(),
		dstMAC:  p.ethernet.DstMAC,
		outPort: p.egress.Number(),
	}
	if err := r.setFlow(param); err != nil {
		return err
	}

	// Send this ethernet packet directly to the destination node
	logger.Debugf("sending a packet (Src=%v, Dst=%v) to egress port %v..", p.ethernet.SrcMAC, p.ethernet.DstMAC, p.egress.ID())
	return r.PacketOut(p.egress, p.rawPacket)
}

func (r *L2Switch) OnPacketIn(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet) error {
	drop, err := r.processPacket(finder, ingress, eth)
	if drop || err != nil {
		return err
	}

	return r.BaseProcessor.OnPacketIn(finder, ingress, eth)
}

func (r *L2Switch) processPacket(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet) (drop bool, err error) {
	logger.Debugf("PACKET_IN.. Ingress=%v, SrcMAC=%v, DstMAC=%v", ingress.ID(), eth.SrcMAC, eth.DstMAC)

	// Drop the some initial PACKET_IN packets on start up so that we can process the high
	// priority packets such as LLDP and ARP probes during initial device and topology setup.
	ready, initTime := ingress.Device().IsReady()
	if ready == false || time.Now().Sub(initTime) < 5*time.Second {
		logger.Debugf("ignoring PACKET_IN: initial time buffering: ingress=%v, initTime=%v", ingress.ID(), initTime)
		// Drop the incoming packet.
		return true, nil
	}

	packet, err := eth.MarshalBinary()
	if err != nil {
		return false, err
	}

	// Broadcast?
	if isBroadcast(eth) {
		logger.Debugf("broadcasting.. SrcMAC=%v, DstMAC=%v", eth.SrcMAC, eth.DstMAC)
		return true, r.stormCtrl.broadcast(ingress, packet)
	}

	logger.Debugf("finding node for %v...", eth.DstMAC)
	dstNode, status, err := finder.Node(eth.DstMAC)
	if err != nil {
		return true, errors.Wrap(err, fmt.Sprintf("locating a node (MAC=%v)", eth.DstMAC))
	}
	if status != network.LocationDiscovered {
		if status == network.LocationUndiscovered {
			// Broadcast!
			logger.Debugf("undiscovered node! broadcasting.. SrcMAC=%v, DstMAC=%v", eth.SrcMAC, eth.DstMAC)
			return true, ingress.Device().Flood(ingress, packet)
		} else if status == network.LocationUnregistered {
			// Drop!
			logger.Debugf("unknown node! dropping.. SrcMAC=%v, DstMAC=%v", eth.SrcMAC, eth.DstMAC)
			return true, nil
		} else {
			panic(fmt.Sprintf("unexpected location status: %v", status))
		}
	}
	logger.Debugf("found the node for %v: deviceID=%v, portNum=%v", eth.DstMAC, dstNode.Port().Device().ID(), dstNode.Port().Number())

	// Disconnected node?
	port := dstNode.Port().Value()
	if port.IsPortDown() || port.IsLinkDown() {
		logger.Debugf("disconnected node! dropping.. SrcMAC=%v, DstMAC=%v", eth.SrcMAC, eth.DstMAC)
		return true, nil
	}

	param := switchParam{}
	// Check whether src and dst nodes reside on a same switch device
	if ingress.Device().ID() == dstNode.Port().Device().ID() {
		param = switchParam{
			finder:    finder,
			ethernet:  eth,
			ingress:   ingress,
			egress:    dstNode.Port(),
			rawPacket: packet,
		}
	} else {
		path := finder.Path(ingress.Device().ID(), dstNode.Port().Device().ID())
		if len(path) == 0 {
			logger.Debugf("empty path.. dropping SrcMAC=%v, DstMAC=%v", eth.SrcMAC, eth.DstMAC)
			return true, nil
		}
		egress := path[0][0]
		// Drop this packet if it goes back to the ingress port to avoid duplicated packet routing
		if ingress.Number() == egress.Number() {
			logger.Debugf("ignore routing path that goes back to the ingress port (SrcMAC=%v, DstMAC=%v)", eth.SrcMAC, eth.DstMAC)
			return true, nil
		}

		param = switchParam{
			finder:    finder,
			ethernet:  eth,
			ingress:   ingress,
			egress:    egress,
			rawPacket: packet,
		}
	}

	return true, r.switching(param)
}

func (r *L2Switch) OnTopologyChange(finder network.Finder) error {
	logger.Debug("OnTopologyChange..")

	// We should remove all edges from all switch devices when the network topology is changed.
	// Otherwise, installed flow rules in switches may result in incorrect packet routing based on the previous topology.
	if err := r.removeAllFlows(finder.Devices()); err != nil {
		return err
	}

	return r.BaseProcessor.OnTopologyChange(finder)
}

func (r *L2Switch) removeAllFlows(devices []*network.Device) error {
	logger.Debug("removing all flows from all devices..")

	for _, d := range devices {
		if d.IsClosed() {
			continue
		}
		if err := d.RemoveFlows(); err != nil {
			return err
		}
		logger.Debugf("removed all flows from DPID %v", d.ID())
	}

	return nil
}

func (r *L2Switch) String() string {
	return fmt.Sprintf("%v", r.Name())
}

func (r *L2Switch) OnPortDown(finder network.Finder, port *network.Port) error {
	logger.Warningf("port down! DPID=%v, number=%v", port.Device().ID(), port.Number())

	device := port.Device()
	factory := device.Factory()
	// Wildcard match
	match, err := factory.NewMatch()
	if err != nil {
		return err
	}
	outPort := openflow.NewOutPort()
	outPort.SetValue(port.Number())

	if err := device.RemoveFlow(match, outPort); err != nil {
		return errors.Wrap(err, fmt.Sprintf("removing flows heading to port %v", port.ID()))
	}
	logger.Debugf("removed all flows heading to the port %v", port.ID())

	return r.BaseProcessor.OnPortDown(finder, port)
}

func (r *L2Switch) OnDeviceUp(finder network.Finder, device *network.Device) error {
	// Make sure that there is only one flow manager in this application.
	r.once.Do(func() {
		// Run the background flow manager.
		go r.flowManager(finder)
	})

	return r.BaseProcessor.OnDeviceUp(finder, device)
}

func (r *L2Switch) flowManager(finder network.Finder) {
	logger.Debug("executed flow manager")

	ticker := time.Tick(1 * time.Minute)
	// Infinite loop.
	for range ticker {
		mac, err := r.db.MACAddrs()
		if err != nil {
			logger.Errorf("failed to get MAC addresses: %v", err)
			continue
		}
		logger.Debugf("got %v MAC addresses", len(mac))

		for _, addr := range mac {
			logger.Debugf("modifying the flow for %v...", addr)
			r.modifyFlows(finder, addr)
		}
	}
}

func (r *L2Switch) modifyFlows(finder network.Finder, mac net.HardwareAddr) {
	// Locate the destination node for the address.
	node, status, err := finder.Node(mac)
	if err != nil {
		logger.Errorf("failed to locate the node %v: %v", mac, err)
		return
	}
	if status != network.LocationDiscovered {
		logger.Debugf("skip flow management for %v: undiscovered location", mac)
		return
	}

	// Disconnected node?
	port := node.Port().Value()
	if port.IsPortDown() || port.IsLinkDown() {
		logger.Debugf("skip flow management for %v: link down", mac)
		return
	}

	// Update the flows on all devices.
	for _, device := range finder.Devices() {
		var egress *network.Port

		// Reside on this device?
		if device.ID() == node.Port().Device().ID() {
			logger.Debugf("reside on the same device: DPID=%v, Port=%v", device.ID(), node.Port().Number())
			egress = node.Port()
		} else {
			// Find the shortest path from this device to an another device that is connected to the destination node.
			path := finder.Path(device.ID(), node.Port().Device().ID())
			// No path to the destination node?
			if len(path) == 0 {
				logger.Debugf("skip flow management for %v on %v: no path", mac, device.ID())
				continue
			}
			egress = path[0][0]
		}

		flow := flowParam{
			device:  device,
			dstMAC:  mac,
			outPort: egress.Number(),
		}
		if err := r.setFlow(flow); err != nil {
			logger.Errorf("failed to modify the flows for %v on %v: %v", mac, device.ID(), err)
			continue
		}
	}
}
