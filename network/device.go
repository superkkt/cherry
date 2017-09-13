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
	"encoding"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/superkkt/cherry/openflow"
	"github.com/superkkt/cherry/openflow/transceiver"
	"github.com/superkkt/cherry/protocol"

	"github.com/superkkt/viper"
)

type Descriptions struct {
	Manufacturer string
	Hardware     string
	Software     string
	Serial       string
	Description  string
}

type Features struct {
	DPID       uint64
	NumBuffers uint32
	NumTables  uint8
}

type Device struct {
	mutex        sync.RWMutex
	id           string
	session      *session
	descriptions Descriptions
	features     Features
	ports        map[uint32]*Port
	flowTableID  uint8 // Table IDs that we install flows
	factory      openflow.Factory
	closed       bool
	flowCache    *flowCache
	vlanID       uint16
}

var (
	ErrClosedDevice = errors.New("already closed device")
)

func newDevice(s *session) *Device {
	if s == nil {
		panic("Session is nil")
	}

	vlanID := viper.GetInt("default.vlan_id")
	if vlanID < 0 || vlanID > 4095 {
		// vlanID should be already checked in the main code.
		panic("invalid default.vlan_id in the config file")
	}

	return &Device{
		session:   s,
		ports:     make(map[uint32]*Port),
		flowCache: newFlowCache(5 * time.Second),
		vlanID:    uint16(vlanID),
	}
}

func (r *Device) String() string {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	v := fmt.Sprintf("Device ID=%v, Descriptions=%+v, Features=%+v, # of ports=%v, FlowTableID=%v, Connected=%v\n", r.id, r.descriptions, r.features, len(r.ports), r.flowTableID, !r.closed)
	for _, p := range r.ports {
		v += fmt.Sprintf("\t%v\n", p.String())
	}

	return v
}

func (r *Device) ID() string {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.id
}

func (r *Device) setID(id string) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.id = id
}

func (r *Device) isReady() bool {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.id) > 0
}

func (r *Device) Factory() openflow.Factory {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.factory
}

func (r *Device) setFactory(f openflow.Factory) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if f == nil {
		panic("Factory is nil")
	}
	r.factory = f
}

func (r *Device) Writer() transceiver.Writer {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.session
}

func (r *Device) Descriptions() Descriptions {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.descriptions
}

func (r *Device) setDescriptions(d Descriptions) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.descriptions = d
}

func (r *Device) Features() Features {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.features
}

func (r *Device) setFeatures(f Features) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.features = f
}

// Port may return nil if there is no port whose number is num
func (r *Device) Port(num uint32) *Port {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.ports[num]
}

func (r *Device) Ports() []*Port {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	p := make([]*Port, 0)
	for _, v := range r.ports {
		p = append(p, v)
	}

	return p
}

func (r *Device) setPort(num uint32, p openflow.Port) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if p == nil {
		panic("Port is nil")
	}
	logger.Debugf("Device=%v, PortNum=%v, AdminUp=%v, LinkUp=%v", r.id, p.Number(), !p.IsPortDown(), !p.IsLinkDown())

	port, ok := r.ports[num]
	if ok {
		port.SetValue(p)
	} else {
		v := NewPort(r, num)
		v.SetValue(p)
		r.ports[num] = v
	}
}

func (r *Device) FlowTableID() uint8 {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.flowTableID
}

func (r *Device) setFlowTableID(id uint8) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.flowTableID = id
}

func (r *Device) SendMessage(msg encoding.BinaryMarshaler) error {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if msg == nil {
		panic("Message is nil")
	}
	if r.closed {
		return ErrClosedDevice
	}

	return r.session.Write(msg)
}

func (r *Device) IsClosed() bool {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.closed
}

// SetFlow installs a normal flow entry for packet switching and routing into the switch device.
func (r *Device) SetFlow(match openflow.Match, port openflow.OutPort) error {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.closed {
		return ErrClosedDevice
	}

	// Set the default VLAN ID. It is necessary to use the L2 MAC flow table of Dell SXXX switches.
	match.SetVLANID(r.vlanID)

	action, err := r.factory.NewAction()
	if err != nil {
		return err
	}
	action.SetOutPort(port)

	inst, err := r.factory.NewInstruction()
	if err != nil {
		return err
	}
	inst.ApplyAction(action)

	// For valid (non-overlapping) ADD requests, or those with no overlap checking,
	// the switch must insert the flow entry at the lowest numbered table for which
	// the switch supports all wildcards set in the flow_match struct, and for which
	// the priority would be observed during the matching process. If a flow entry
	// with identical header fields and priority already resides in any table, then
	// that entry, including its counters, must be removed, and the new flow entry added.
	flow, err := r.factory.NewFlowMod(openflow.FlowAdd)
	if err != nil {
		return err
	}
	flow.SetTableID(r.flowTableID)
	// This idle timeout is actually useless because we update the installed flows
	// more frequently than this timeout.
	flow.SetIdleTimeout(90)
	flow.SetPriority(10)
	flow.SetFlowMatch(match)
	flow.SetFlowInstruction(inst)

	ok, err := r.flowCache.InProgress(match, port)
	if err != nil {
		return err
	}
	if ok {
		logger.Debugf("skip to install a new flow: already installed one: deviceID=%v", r.id)
		return nil
	}
	// Install the new flow.
	if err := r.session.Write(flow); err != nil {
		return err
	}
	if err := r.flowCache.Add(match, port); err != nil {
		return err
	}

	barrier, err := r.factory.NewBarrierRequest()
	if err != nil {
		return err
	}

	return r.session.Write(barrier)
}

// RemoveFlows removes all the normal flows except special ones for table miss and ARP packets.
func (r *Device) RemoveFlows() error {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.closed {
		return ErrClosedDevice
	}

	match, err := r.factory.NewMatch()
	if err != nil {
		return err
	}
	// Default VLAN ID specified for the normal flows.
	match.SetVLANID(r.vlanID)

	// Set output port to OFPP_NONE
	port := openflow.NewOutPort()
	port.SetNone()

	flowmod, err := r.factory.NewFlowMod(openflow.FlowDelete)
	if err != nil {
		return err
	}
	// Remove all the normal flows, except the special table miss and ARP flows whose MSB is 1.
	flowmod.SetCookieMask(0x1 << 63)
	flowmod.SetTableID(0xFF) // ALL
	flowmod.SetFlowMatch(match)
	flowmod.SetOutPort(port)
	if err := r.session.Write(flowmod); err != nil {
		return err
	}
	r.flowCache.RemoveAll()

	return nil
}

// TODO:
// Remove the flow caches that match the removed flows. This is not a critical
// issue, but same flows cannot be installed until the caches are expired.
func (r *Device) RemoveFlow(match openflow.Match, port openflow.OutPort) error {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.closed {
		return ErrClosedDevice
	}

	// Default VLAN ID specified for the normal flows.
	match.SetVLANID(r.vlanID)

	flowmod, err := r.factory.NewFlowMod(openflow.FlowDelete)
	if err != nil {
		return err
	}
	// Remove all the normal flows, except the table miss and ARP flows whose MSB is 1.
	flowmod.SetCookieMask(0x1 << 63)
	flowmod.SetTableID(0xFF) // ALL
	flowmod.SetFlowMatch(match)
	flowmod.SetOutPort(port)

	return r.session.Write(flowmod)
}

// TODO:
// Remove the flow caches that match the removed flows. This is not a critical
// issue, but same flows cannot be installed until the caches are expired.
func (r *Device) RemoveFlowByMAC(mac net.HardwareAddr) error {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.closed {
		return ErrClosedDevice
	}

	match, err := r.factory.NewMatch()
	if err != nil {
		return err
	}
	// Default VLAN ID specified for the normal flows.
	match.SetVLANID(r.vlanID)
	match.SetDstMAC(mac)

	port := openflow.NewOutPort()
	port.SetNone()

	flowmod, err := r.factory.NewFlowMod(openflow.FlowDelete)
	if err != nil {
		return err
	}
	// Remove all the normal flows, except the table miss and ARP flows whose MSB is 1.
	flowmod.SetCookieMask(0x1 << 63)
	flowmod.SetTableID(0xFF) // ALL
	flowmod.SetFlowMatch(match)
	flowmod.SetOutPort(port)

	return r.session.Write(flowmod)
}

func makeARPAnnouncement(ip net.IP, mac net.HardwareAddr) ([]byte, error) {
	v := protocol.NewARPRequest(mac, ip, ip)
	anon, err := v.MarshalBinary()
	if err != nil {
		return nil, err
	}
	eth := protocol.Ethernet{
		SrcMAC:  mac,
		DstMAC:  net.HardwareAddr([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}),
		Type:    0x0806,
		Payload: anon,
	}

	return eth.MarshalBinary()
}

func (r *Device) SendARPAnnouncement(ip net.IP, mac net.HardwareAddr) error {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.closed {
		return ErrClosedDevice
	}

	announcement, err := makeARPAnnouncement(ip, mac)
	if err != nil {
		return err
	}

	return r.flood(nil, announcement)
}

func (r *Device) SendARPProbe(sha net.HardwareAddr, tpa net.IP) error {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.closed {
		return ErrClosedDevice
	}

	probe, err := makeARPProbe(sha, tpa)
	if err != nil {
		return err
	}

	return r.flood(nil, probe)
}

// https://en.wikipedia.org/wiki/Address_Resolution_Protocol#ARP_probe
//
// An ARP probe is an ARP request constructed with an all-zero sender IP address (SPA).
// The term is used in the IPv4 Address Conflict Detection specification (RFC 5227).
// Before beginning to use an IPv4 address (whether received from manual configuration,
// DHCP, or some other means), a host implementing this specification must test to see
// if the address is already in use, by broadcasting ARP probe packets.
func makeARPProbe(sha net.HardwareAddr, tpa net.IP) ([]byte, error) {
	arp := protocol.NewARPRequest(sha, net.IPv4(0, 0, 0, 0), tpa)
	probe, err := arp.MarshalBinary()
	if err != nil {
		return nil, err
	}
	eth := protocol.Ethernet{
		SrcMAC:  sha,
		DstMAC:  net.HardwareAddr([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}),
		Type:    0x0806,
		Payload: probe,
	}

	return eth.MarshalBinary()
}

// Flood broadcasts the packet to all ports of this device, except the ingress port if ingress is not nil.
func (r *Device) Flood(ingress *Port, packet []byte) error {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.closed {
		return ErrClosedDevice
	}

	return r.flood(ingress, packet)
}

// flood broadcasts the packet to all ports of this device, except the ingress port if ingress is not nil.
func (r *Device) flood(ingress *Port, packet []byte) error {
	inPort := openflow.NewInPort()
	if ingress != nil {
		inPort.SetValue(ingress.Number())
	} else {
		inPort.SetController()
	}

	outPort := openflow.NewOutPort()
	// FLOOD means all ports except the ingress one.
	outPort.SetFlood()

	action, err := r.factory.NewAction()
	if err != nil {
		return err
	}
	action.SetOutPort(outPort)

	out, err := r.factory.NewPacketOut()
	if err != nil {
		return err
	}
	out.SetInPort(inPort)
	out.SetAction(action)
	out.SetData(packet)

	return r.session.Write(out)
}

func (r *Device) Close() {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.closed = true
}
