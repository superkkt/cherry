/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package device

import (
	"git.sds.co.kr/cherry.git/cherryd/internal/graph"
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
		mac:   make(map[string]Point),
		point: make(map[string][]net.HardwareAddr),
	}
}

func (r *Topology) add(dpid uint64, d *Device) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.pool[dpid] = d
	r.graph.AddVertex(d)
}

func (r *Topology) remove(dpid uint64) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	device, ok := r.pool[dpid]
	if !ok {
		return
	}
	r.graph.RemoveVertex(device)
	delete(r.pool, dpid)
}

func (r *Topology) Get(dpid uint64) *Device {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.pool[dpid]
}

type HostPool struct {
	mutex sync.Mutex
	mac   map[string]Point
	point map[string][]net.HardwareAddr
}

func (r *HostPool) add(mac net.HardwareAddr, p Point) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if mac == nil || p.Node == nil {
		panic("nil parameters")
	}

	// Check duplication
	prev, ok := r.mac[mac.String()]
	if ok {
		if prev.Compare(p) {
			// Do nothing if same one already exists
			return
		}
		// Remove previous values
		r._remove(prev)
	}

	r.mac[mac.String()] = p

	v, ok := r.point[p.ID()]
	if ok {
		v = append(v, mac)
	} else {
		v = []net.HardwareAddr{mac}
	}
	r.point[p.ID()] = v
}

func (r *HostPool) _remove(p Point) {
	v, ok := r.point[p.ID()]
	if !ok {
		return
	}
	for _, mac := range v {
		delete(r.mac, mac.String())
	}
	delete(r.point, p.ID())
}

func (r *HostPool) remove(p Point) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if p.Node == nil {
		panic("nil parameter")
	}

	r._remove(p)
}

func (r *HostPool) Find(mac net.HardwareAddr) (p Point, ok bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if mac == nil {
		panic("nil MAC address")
	}
	p, ok = r.mac[mac.String()]
	return
}
