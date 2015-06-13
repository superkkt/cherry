/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package controller

import (
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"net"
	"sync"
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
	log          log.Logger
	watcher      Watcher
	controllers  map[uint]*Controller
	descriptions Descriptions
	features     Features
	ports        map[uint]*Port
	flowTableID  uint8 // Table IDs that we install flows
	auxID        uint
}

func NewDevice(id string, log log.Logger, w Watcher) *Device {
	return &Device{
		id:          id,
		log:         log,
		watcher:     w,
		controllers: make(map[uint]*Controller),
		ports:       make(map[uint]*Port),
	}
}

func (r *Device) ID() string {
	return r.id
}

func (r *Device) addConn(c net.Conn) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if c == nil {
		panic("nil connection")
	}

	ctr := NewController(r, c, r.log)
	r.controllers[r.auxID] = ctr
	go func(id uint) {
		defer c.Close()
		ctr.Run()
		r.disconnected(id)
	}(r.auxID)
	r.auxID++
}

func (r *Device) disconnected(id uint) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.controllers, id)
	// We have no controllers?
	if len(r.controllers) == 0 {
		r.watcher.DeviceRemoved(r.id)
	}
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
func (r *Device) Port(num uint) *Port {
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

// A caller should make sure the mutex is locked before calling this function
func (r *Device) setPort(num uint, p openflow.Port) {
	port := NewPort(r, num)
	port.SetValue(p)
	r.ports[num] = port
}

func (r *Device) addPort(num uint, p openflow.Port) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.setPort(num, p)
}

func (r *Device) updatePort(num uint, p openflow.Port) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()
	defer r.setPort(num, p)

	port := r.Port(num)
	if port == nil {
		return
	}

	if p.IsPortDown() || p.IsLinkDown() {
		r.watcher.PortRemoved(port)
	} else {
		// TODO: Send LLDP
	}

	// TODO: Send this event to a watcher
}

func (r *Device) FlowTableID() uint8 {
	return r.flowTableID
}

func (r *Device) setFlowTableID(id uint8) {
	r.flowTableID = id
}
