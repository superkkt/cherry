/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
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
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/graph"
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
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
	Node(mac net.HardwareAddr) *Node
	Path(srcDeviceID, dstDeviceID string) [][2]*Port
}

type topology struct {
	mutex sync.RWMutex
	// Key is IP address of a device
	devices map[string]*Device
	// Key is MAC address of a node
	nodes    map[string]*Node
	log      log.Logger
	graph    *graph.Graph
	listener TopologyEventListener
}

func newTopology(log log.Logger) *topology {
	return &topology{
		devices: make(map[string]*Device),
		nodes:   make(map[string]*Node),
		log:     log,
		graph:   graph.New(),
	}
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
		r.log.Err(fmt.Sprintf("topology: executing OnTopologyChange: %v", err))
		return
	}
}

func (r *topology) Devices() []*Device {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	v := make([]*Device, 0)
	for _, d := range r.devices {
		v = append(v, d)
	}

	return v
}

// Device may return nil if a device whose ID is id does not exist
func (r *topology) Device(id string) *Device {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.devices[id]
}

func (r *topology) DeviceAdded(d *Device) {
	// Write lock
	r.mutex.Lock()
	r.devices[d.ID()] = d
	// Unlock
	r.mutex.Unlock()

	r.graph.AddVertex(d)
	r.sendEvent()
}

func (r *topology) removeDevice(d *Device) {
	id := d.ID()
	// Device exists?
	d, ok := r.devices[id]
	if !ok {
		return
	}
	// Remove all nodes connected to this device
	r.removeAllNodes(d)
	// Remove from the device database
	delete(r.devices, id)
}

func (r *topology) removeAllNodes(d *Device) {
	ports := d.Ports()
	for _, p := range ports {
		for _, n := range p.Nodes() {
			delete(r.nodes, n.MAC().String())
			p.removeNode(n.MAC())
		}
	}
}

func (r *topology) DeviceRemoved(d *Device) {
	// Write lock
	r.mutex.Lock()
	r.removeDevice(d)
	// Unlock
	r.mutex.Unlock()

	// Remove from the network topology
	r.graph.RemoveVertex(d)
	r.sendEvent()
}

func (r *topology) DeviceLinked(ports [2]*Port) {
	link := newLink(ports)
	if err := r.graph.AddEdge(link); err != nil {
		r.log.Err(fmt.Sprintf("topology: %v", err))
		return
	}
	r.sendEvent()
}

func (r *topology) NodeAdded(n *Node) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if n == nil {
		panic("node is nil")
	}

	node, ok := r.nodes[n.MAC().String()]
	// Do we already have a port that has this node?
	if ok {
		// Remove this node from the port
		port := node.Port()
		port.removeNode(node.MAC())
	}
	// Add new node
	r.nodes[n.MAC().String()] = n
}

// Node may return nil if a node whose MAC is mac does not exist
func (r *topology) Node(mac net.HardwareAddr) *Node {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.nodes[mac.String()]
}

func (r *topology) PortRemoved(p *Port) {
	// Write lock
	r.mutex.Lock()
	// Remove hosts connected to the port from the host database
	for _, v := range p.Nodes() {
		delete(r.nodes, v.MAC().String())
		p.removeNode(v.MAC())
	}
	// Unlock
	r.mutex.Unlock()

	if r.graph.IsEdge(p) {
		// Remove an edge from the graph if this port is an edge connected to another switch
		r.graph.RemoveEdge(p)
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

	path := r.graph.FindPath(src, dst)
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
