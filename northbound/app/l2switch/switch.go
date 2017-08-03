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
	"math"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/superkkt/cherry/network"
	"github.com/superkkt/cherry/northbound/app"
	"github.com/superkkt/cherry/openflow"
	"github.com/superkkt/cherry/protocol"

	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/superkkt/go-logging"
	"github.com/superkkt/viper"
)

var (
	logger = logging.MustGetLogger("l2switch")
)

type L2Switch struct {
	app.BaseProcessor
	vlanID    uint16
	cache     *flowCache
	stormCtrl *stormController
	db        Database
	// Idle timeout in second.
	idleTimeout uint16
	// Hard timeout in second.
	hardTimeout uint16
}

type flowCache struct {
	cache *lru.Cache
}

func newFlowCache() *flowCache {
	c, err := lru.New(8192)
	if err != nil {
		panic(fmt.Sprintf("LRU flow cache: %v", err))
	}

	return &flowCache{
		cache: c,
	}
}

func (r *flowCache) getKeyString(flow flowParam) string {
	return fmt.Sprintf("%v/%v/%v", flow.device.ID(), flow.dstMAC, flow.outPort)
}

func (r *flowCache) exist(flow flowParam) bool {
	v, ok := r.cache.Get(r.getKeyString(flow))
	if !ok {
		return false
	}
	// Timeout?
	if time.Since(v.(time.Time)) > 5*time.Second {
		return false
	}

	return true
}

func (r *flowCache) add(flow flowParam) {
	// Update if the key already exists
	r.cache.Add(r.getKeyString(flow), time.Now())
}

type Database interface {
	// AddFlow adds a new flow into the database and returns its unique ID.
	AddFlow(swDPID uint64, dstMAC net.HardwareAddr, outPort uint32) (flowID uint64, err error)

	// RemoveFlow removes the flow specified by flowID from the database.
	RemoveFlow(flowID uint64) error

	// RemoveFlows remove all the flows that belong to the device whose ID is swDPID.
	RemoveFlows(swDPID uint64) error
}

func New(db Database) *L2Switch {
	return &L2Switch{
		cache:     newFlowCache(),
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
	vlanID := viper.GetInt("flow.vlan_id")
	if vlanID < 0 || vlanID > 4095 {
		return errors.New("invalid flow.vlan_id in the config file")
	}
	r.vlanID = uint16(vlanID)

	idleTimeout := viper.GetInt("flow.idle_timeout")
	if idleTimeout <= 0 || idleTimeout > math.MaxUint16 {
		return errors.New("invalid flow.idle_timeout in the config file")
	}
	r.idleTimeout = uint16(idleTimeout)

	hardTimeout := viper.GetInt("flow.hard_timeout")
	if hardTimeout < 0 || hardTimeout > math.MaxUint16 {
		return errors.New("invalid flow.hard_timeout in the config file")
	}
	if hardTimeout > 0 && hardTimeout <= idleTimeout*2 {
		return fmt.Errorf("flow.hard_timeout should be greater than %v seconds", idleTimeout*2)
	}
	r.hardTimeout = uint16(hardTimeout)

	return nil
}

func (r *L2Switch) Name() string {
	return "L2Switch"
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

func (r *flowParam) String() string {
	return fmt.Sprintf("Device=%v, EtherType=%v, InPort=%v, OutPort=%v, SrcMAC=%v, DstMAC=%v",
		r.device.ID(), r.etherType, r.inPort, r.outPort, r.srcMAC, r.dstMAC)
}

func (r *L2Switch) installFlow(p flowParam) error {
	// Skip the installation if p is already installed
	if r.cache.exist(p) {
		logger.Debugf("skipping duplicated flow installation: deviceID=%v, dstMAC=%v, outPort=%v",
			p.device.ID(), p.dstMAC, p.outPort)
		return nil
	}

	f := p.device.Factory()
	match, err := f.NewMatch()
	if err != nil {
		return err
	}
	match.SetVLANID(r.vlanID)
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
	flow.SetCookie(r.getFlowID(p))
	flow.SetTableID(p.device.FlowTableID())
	flow.SetIdleTimeout(r.idleTimeout)
	if r.hardTimeout > 0 {
		// Extra random interval in order to avoid a lot of timed-out flows at once.
		padding := uint16(rand.Intn(5))
		flow.SetHardTimeout(r.hardTimeout + padding)
	}
	flow.SetPriority(10)
	flow.SetFlowMatch(match)
	flow.SetFlowInstruction(inst)

	if err := p.device.SendMessage(flow); err != nil {
		return err
	}
	barrier, err := f.NewBarrierRequest()
	if err != nil {
		return err
	}
	if err := p.device.SendMessage(barrier); err != nil {
		return err
	}
	logger.Debugf("installed a flow rule: %+v", p)

	r.cache.add(p)
	logger.Debugf("added a flow cache entry: deviceID=%v, dstMAC=%v, outPort=%v", p.device.ID(), p.dstMAC, p.outPort)

	return nil
}

func (r *L2Switch) getFlowID(p flowParam) uint64 {
	dpid, err := strconv.ParseUint(p.device.ID(), 10, 64)
	if err != nil {
		logger.Errorf("failed to parse the switch DPID: %v", err)
		// Fallback.
		return 0
	}

	flowID, err := r.db.AddFlow(dpid, p.dstMAC, p.outPort)
	if err != nil {
		logger.Errorf("failed to add a new flow: %v", err)
		// Fallback.
		return 0
	}

	return flowID
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
		device:    p.ingress.Device(),
		etherType: p.ethernet.Type,
		inPort:    p.ingress.Number(),
		outPort:   p.egress.Number(),
		srcMAC:    p.ethernet.SrcMAC,
		dstMAC:    p.ethernet.DstMAC,
	}
	if err := r.installFlow(param); err != nil {
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
		if err := d.RemoveAllFlows(); err != nil {
			return err
		}
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

func (r *L2Switch) OnFlowRemoved(finder network.Finder, flow openflow.FlowRemoved) error {
	if err := r.db.RemoveFlow(flow.Cookie()); err != nil {
		logger.Errorf("failed to remove a flow: %v", err)
		// Ignore this error and keep go on.
	}

	return r.BaseProcessor.OnFlowRemoved(finder, flow)
}

func (r *L2Switch) OnDeviceDown(finder network.Finder, device *network.Device) error {
	dpid, err := strconv.ParseUint(device.ID(), 10, 64)
	if err != nil {
		logger.Errorf("failed to parse the device ID: %v", device.ID())
		return r.BaseProcessor.OnDeviceDown(finder, device)
	}

	if err := r.db.RemoveFlows(dpid); err != nil {
		logger.Errorf("failed to remove all flow histories: %v", err)
		return r.BaseProcessor.OnDeviceDown(finder, device)
	}
	logger.Debugf("removed all the flow histories for DPID %v", device.ID())

	return r.BaseProcessor.OnDeviceDown(finder, device)
}
