/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package controller

import (
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/graph"
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"net"
	"sync"
)

type Watcher interface {
	DeviceLinked(ports [2]*Port)
	DeviceRemoved(id string)
	NodeAdded(n *Node)
	PortRemoved(p *Port)
}

type Topology struct {
	mutex sync.RWMutex
	// Key is IP address of a device
	devices map[string]*Device
	// Key is MAC address of a node
	nodes map[string]*Node
	log   log.Logger
	graph *graph.Graph
}

func NewTopology(log log.Logger) *Topology {
	return &Topology{
		devices: make(map[string]*Device),
		nodes:   make(map[string]*Node),
		log:     log,
		graph:   graph.New(),
	}
}

// Caller should make sure the mutex is locked before calling this function
func (r *Topology) addDevice(id string, d *Device) {
	if d == nil {
		panic("nil device")
	}
	r.devices[id] = d
	r.graph.AddVertex(d)
}

// Device may return nil if a device whose ID is id does not exist
func (r *Topology) Device(id string) *Device {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.devices[id]
}

func (r *Topology) removeAllNodes(d *Device) {
	ports := d.Ports()
	for _, p := range ports {
		for _, n := range p.Nodes() {
			delete(r.nodes, n.MAC().String())
			p.RemoveNode(n.MAC())
		}
	}
}

func (r *Topology) DeviceRemoved(id string) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Device exists?
	d, ok := r.devices[id]
	if !ok {
		return
	}
	// Remove all nodes connected to this device
	r.removeAllNodes(d)
	// Remove from the network topology
	r.graph.RemoveVertex(d)
	// Remove from the device database
	delete(r.devices, id)
}

func (r *Topology) DeviceLinked(ports [2]*Port) {
	link := NewLink(ports)
	if err := r.graph.AddEdge(link); err != nil {
		r.log.Err(fmt.Sprintf("DeviceLinked: %v", err))
		return
	}
}

func (r *Topology) NodeAdded(n *Node) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if n == nil {
		panic("node is nil")
	}

	node, ok := r.nodes[n.MAC().String()]
	// Do we already have a port that has this node?
	if ok {
		// Remove the node from the port
		port := node.Port()
		port.RemoveNode(node.MAC())
	}
	// Add new node
	r.nodes[n.MAC().String()] = n
}

// Node may return nil if a node whose MAC is mac does not exist
func (r *Topology) Node(mac net.HardwareAddr) *Node {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.nodes[mac.String()]
}

func (r *Topology) PortRemoved(p *Port) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Remove hosts connected to the port from the host database
	for _, v := range p.Nodes() {
		delete(r.nodes, v.MAC().String())
		p.RemoveNode(v.MAC())
	}
	// Remove an edge from the graph if this port is an edge connected to another switch
	r.graph.RemoveEdge(p)
}

func (r *Topology) AddDeviceConn(c net.Conn) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	id := c.RemoteAddr().String()
	// Do we already have the device whose source IP address is id?
	d, ok := r.devices[id]
	if !ok {
		d = NewDevice(id, r.log, r)
	}
	d.addConn(c)
	if !ok {
		r.addDevice(id, d)
	}
}

// TODO: Path()
