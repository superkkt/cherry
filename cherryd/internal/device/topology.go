/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package device

import (
	"git.sds.co.kr/cherry.git/cherryd/graph"
	"net"
	"sync"
)

var Switches *Topology
var Hosts *HostPool

type Topology struct {
	mutex sync.Mutex
	pool  map[uint64]*Device
	graph *graph.Graph
}

func init() {
	Switches = &Topology{
		pool:  make(map[uint64]*Device),
		graph: graph.New(),
	}
	Hosts = &HostPool{
		pool: make(map[string]Connection),
	}
}

func (r *Topology) add(dpid uint64, d *Device) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.pool[dpid] = d
	r.graph.AddVertex(d)
	r.graph.CalculateMST()
}

func (r *Topology) remove(dpid uint64) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	device, ok := r.pool[dpid]
	if !ok {
		return
	}
	r.graph.RemoveVertex(device)
	r.graph.CalculateMST()
	delete(r.pool, dpid)
}

func (r *Topology) Get(dpid uint64) *Device {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.pool[dpid]
}

func (r *Topology) link(e *Edge) {
	r.graph.AddEdge(e)
	r.graph.CalculateMST()
}

func (r *Topology) unlink(e *Edge) {
	r.graph.RemoveEdge(e)
	r.graph.CalculateMST()
}

type Connection struct {
	Device *Device
	Port   uint32
}

type HostPool struct {
	mutex sync.Mutex
	pool  map[string]Connection
}

func (r *HostPool) add(mac net.HardwareAddr, device *Device, port uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if mac == nil {
		panic("nil MAC address")
	}
	r.pool[mac.String()] = Connection{device, port}
}

func (r *HostPool) remove(mac net.HardwareAddr) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if mac == nil {
		panic("nil MAC address")
	}
	delete(r.pool, mac.String())
}

func (r *HostPool) Find(mac net.HardwareAddr) (Connection, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if mac == nil {
		panic("nil MAC address")
	}
	v, ok := r.pool[mac.String()]
	return v, ok
}
