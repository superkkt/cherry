/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package network

import (
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/graph"
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"net"
	"sync"
)

type Watcher interface {
	DeviceAdded(*Device)
	DeviceLinked([2]*Port)
	DeviceRemoved(*Device)
	NodeAdded(*Node)
	PortRemoved(*Port)
}

type Finder interface {
	Device(id string) *Device
	Node(mac net.HardwareAddr) *Node
	// TODO: Path()
	IsDisabledPort(p *Port) bool
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

// Device may return nil if a device whose ID is id does not exist
func (r *Topology) Device(id string) *Device {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.devices[id]
}

// TODO: 정말로 모든 노드가 다 지워졌는지 실제로 개수 찍어보면서 테스트
func (r *Topology) removeAllNodes(d *Device) {
	ports := d.Ports()
	for _, p := range ports {
		for _, n := range p.Nodes() {
			delete(r.nodes, n.MAC().String())
			p.RemoveNode(n.MAC())
		}
	}
}

func (r *Topology) DeviceAdded(d *Device) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.devices[d.ID()] = d
	r.graph.AddVertex(d)
}

// TODO: 디바이스 리스트와 그래프가 정상적으로 갱신되는지 남은 데이터 찍어보면서 테스트
func (r *Topology) DeviceRemoved(d *Device) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	id := d.ID()
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

// TODO: 호스트 리스트와 그래프가 정상적으로 갱신되는지 남은 데이터 찍어보면서 테스트
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

// TODO: Path()

func (r *Topology) IsDisabledPort(p *Port) bool {
	if r.graph.IsEdge(p) && !r.graph.IsEnabledPoint(p) {
		return true
	}

	return false
}
