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
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"net"
	"sync"
	"time"
)

type Port struct {
	mutex     sync.RWMutex
	device    *Device
	number    uint
	value     openflow.Port
	nodes     []*Node
	timestamp time.Time
}

func NewPort(d *Device, num uint) *Port {
	return &Port{
		device: d,
		number: num,
		nodes:  make([]*Node, 0),
	}
}

func (r *Port) ID() string {
	return fmt.Sprintf("%v:%v", r.device.ID(), r.number)
}

func (r *Port) Vertex() graph.Vertex {
	return r.device
}

func (r *Port) Device() *Device {
	return r.device
}

func (r *Port) Value() openflow.Port {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.value
}

func (r *Port) SetValue(p openflow.Port) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.value = p
	r.timestamp = time.Now()
}

// Duration returns the time during which this port activated
func (r *Port) Duration() time.Duration {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.timestamp.IsZero() {
		return time.Duration(0)
	}

	return time.Now().Sub(r.timestamp)
}

func (r *Port) Nodes() []*Node {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.nodes
}

func (r *Port) AddNode(mac net.HardwareAddr) *Node {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	node := NewNode(r, mac)
	r.nodes = append(r.nodes, node)

	return node
}

func (r *Port) RemoveNode(mac net.HardwareAddr) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	nodes := make([]*Node, 0)
	for _, v := range r.nodes {
		if v.MAC().String() == mac.String() {
			continue
		}
		nodes = append(nodes, v)
	}
	r.nodes = nodes
}
