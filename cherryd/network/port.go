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
	"fmt"
	"github.com/superkkt/cherry/cherryd/graph"
	"github.com/superkkt/cherry/cherryd/openflow"
	"net"
	"sync"
	"time"
)

type Port struct {
	mutex     sync.RWMutex
	device    *Device
	number    uint32
	value     openflow.Port
	nodes     map[string]*Node
	timestamp time.Time
}

func NewPort(d *Device, num uint32) *Port {
	return &Port{
		device: d,
		number: num,
		nodes:  make(map[string]*Node),
	}
}

func (r *Port) String() string {
	return fmt.Sprintf("Port Number=%v, Device_id=%v, # of nodes=%v", r.number, r.device.ID(), len(r.nodes))
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

func (r *Port) Number() uint32 {
	return r.number
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
func (r *Port) duration() time.Duration {
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

	v := make([]*Node, 0)
	for _, n := range r.nodes {
		v = append(v, n)
	}

	return v
}

func (r *Port) addNode(mac net.HardwareAddr) *Node {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	node := NewNode(r, mac)
	r.nodes[mac.String()] = node

	return node
}

func (r *Port) removeNode(mac net.HardwareAddr) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.nodes, mac.String())
}
