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
	"bytes"
	"fmt"
	"github.com/superkkt/cherry/cherryd/graph"
	"github.com/superkkt/cherry/cherryd/log"
	"net"
	"sync"
)

type watcher interface {
	DeviceAdded(*Device)
	DeviceLinked([2]*Port)
	DeviceRemoved(*Device)
	NodeAdded(*Node)
	PortRemoved(*Port)
}

type Finder interface {
	Device(id string) *Device
	Devices() []*Device
	// IsEnabledBySTP returns whether p is disabled by spanning tree protocol
	IsEnabledBySTP(p *Port) bool
	// IsEdge returns whether p is an edge among two switches
	IsEdge(p *Port) bool
	Node(device *Device, mac net.HardwareAddr) *Node
	Path(srcDeviceID, dstDeviceID string) [][2]*Port
}

type device struct {
	value *Device
	// Key is MAC address of a node
	nodes map[string]*Node
}

func (r device) String() string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("Device: %v (# of nodes = %v)\n", r.value, len(r.nodes)))
	for _, v := range r.nodes {
		buf.WriteString(fmt.Sprintf("Node: %v\n", v))
	}
	buf.WriteString("\n")

	return buf.String()
}

type topology struct {
	mutex sync.RWMutex
	// Key is the device ID
	devices  map[string]*device
	log      log.Logger
	graph    *graph.Graph
	listener TopologyEventListener
}

func newTopology(log log.Logger) *topology {
	return &topology{
		devices: make(map[string]*device),
		log:     log,
		graph:   graph.New(),
	}
}

func (r *topology) String() string {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var buf bytes.Buffer
	for _, v := range r.devices {
		buf.WriteString(fmt.Sprintf("%v\n", v))
	}
	buf.WriteString(fmt.Sprintf("%v\n", r.graph))

	return buf.String()
}

func (r *topology) setEventListener(l TopologyEventListener) {
	r.listener = l
}

// Caller should make sure the mutex is unlocked before calling this function.
// Otherwise, event listeners may cause a deadlock by calling other topology functions.
func (r *topology) sendEvent() {
	if r.listener == nil {
		return
	}

	if err := r.listener.OnTopologyChange(r); err != nil {
		r.log.Err(fmt.Sprintf("Topology: executing OnTopologyChange: %v", err))
		return
	}
}

func (r *topology) Devices() []*Device {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	v := make([]*Device, 0)
	for _, d := range r.devices {
		v = append(v, d.value)
	}

	return v
}

// Device may return nil if a device whose ID is id does not exist
func (r *topology) Device(id string) *Device {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	d, ok := r.devices[id]
	if !ok {
		return nil
	}

	return d.value
}

func (r *topology) DeviceAdded(d *Device) {
	func() {
		// Write lock
		r.mutex.Lock()
		defer r.mutex.Unlock()

		// Clear all nodes from all devices because STP will be updated
		r.clearNodes()
		r.devices[d.ID()] = &device{
			value: d,
			nodes: make(map[string]*Node),
		}
		r.graph.AddVertex(d)
	}()
	r.sendEvent()
}

// XXX: Caller should lock the mutex
func (r *topology) removeDevice(d *Device) {
	// Device exists?
	_, ok := r.devices[d.ID()]
	if !ok {
		return
	}
	// Remove from the device database
	delete(r.devices, d.ID())
}

// XXX: Caller should lock the mutex
func (r *topology) clearNodes() {
	for _, v := range r.devices {
		v.nodes = make(map[string]*Node)
	}
}

func (r *topology) DeviceRemoved(d *Device) {
	func() {
		// Write lock
		r.mutex.Lock()
		defer r.mutex.Unlock()

		// Clear all nodes from all devices because STP will be updated
		r.clearNodes()
		r.removeDevice(d)
		r.graph.RemoveVertex(d)
	}()
	r.sendEvent()
}

func (r *topology) DeviceLinked(ports [2]*Port) {
	func() {
		// Write lock
		r.mutex.Lock()
		defer r.mutex.Unlock()

		// Clear all nodes from all devices because STP will be updated
		r.clearNodes()
		link := newLink(ports)
		if err := r.graph.AddEdge(link); err != nil {
			r.log.Err(fmt.Sprintf("Topology: adding new graph edge: %v", err))
			return
		}
	}()
	r.sendEvent()
}

func (r *topology) NodeAdded(n *Node) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if n == nil {
		panic("node is nil")
	}

	device, ok := r.devices[n.Port().Device().ID()]
	if !ok {
		panic("A node is added, but there is a no device related with the node!")
	}

	node, ok := device.nodes[n.MAC().String()]
	// Do we already have a port that has this node?
	if ok {
		// Remove this node from the port
		port := node.Port()
		port.removeNode(node.MAC())
	}
	// Add (update) new node
	device.nodes[n.MAC().String()] = n
}

// Node may return nil if a node whose MAC is mac does not exist
func (r *topology) Node(d *Device, mac net.HardwareAddr) *Node {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	device, ok := r.devices[d.ID()]
	if !ok {
		panic(fmt.Sprintf("Unknown device ID: %v", d.ID()))
	}

	return device.nodes[mac.String()]
}

// XXX: Caller should lock the mutex
func (r *topology) removeNode(mac net.HardwareAddr) {
	for _, v := range r.devices {
		n := v.nodes[mac.String()]
		if n == nil {
			continue
		}
		n.Port().removeNode(mac)
		delete(v.nodes, mac.String())
	}
}

func (r *topology) PortRemoved(p *Port) {
	edge := false

	func() {
		// Write lock
		r.mutex.Lock()
		defer r.mutex.Unlock()

		// Remove hosts connected to the port from the host database
		for _, v := range p.Nodes() {
			r.removeNode(v.MAC())
			p.removeNode(v.MAC())
		}

		if edge = r.graph.IsEdge(p); edge == true {
			// Clear all nodes from all devices because STP will be updated
			r.clearNodes()
			// Remove an edge from the graph if this port is an edge connected to another switch
			r.graph.RemoveEdge(p)
		}
	}()

	if edge {
		// XXX: Make sure the mutex is unlocked before calling sendEvent()
		r.sendEvent()
	}
}

func (r *topology) Path(srcDeviceID, dstDeviceID string) [][2]*Port {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	v := make([][2]*Port, 0)
	src := r.devices[srcDeviceID]
	dst := r.devices[dstDeviceID]
	// Unknown source or destination device?
	if src == nil || dst == nil {
		// Return empty path
		return v
	}

	path := r.graph.FindPath(src.value, dst.value)
	for _, p := range path {
		device := p.V.(*Device)
		link := p.E.(*link)
		v = append(v, pickPort(device, link))
	}

	return v
}

func pickPort(d *Device, l *link) [2]*Port {
	p := l.Points()
	if p[0].Vertex().ID() == d.ID() {
		return [2]*Port{p[0].(*Port), p[1].(*Port)}
	}

	return [2]*Port{p[1].(*Port), p[0].(*Port)}
}

func (r *topology) IsEdge(p *Port) bool {
	return r.graph.IsEdge(p)
}

func (r *topology) IsEnabledBySTP(p *Port) bool {
	return r.graph.IsEnabledPoint(p)
}
