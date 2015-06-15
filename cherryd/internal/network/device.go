/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package network

import (
	"encoding"
	"errors"
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"git.sds.co.kr/cherry.git/cherryd/openflow/trans"
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
	finder       Finder
	controllers  map[uint8]trans.Writer
	descriptions Descriptions
	features     Features
	ports        map[uint]*Port
	flowTableID  uint8 // Table IDs that we install flows
}

func NewDevice(id string, log log.Logger, w Watcher, f Finder) *Device {
	return &Device{
		id:          id,
		log:         log,
		watcher:     w,
		finder:      f,
		controllers: make(map[uint8]trans.Writer),
		ports:       make(map[uint]*Port),
	}
}

func (r *Device) ID() string {
	return r.id
}

func (r *Device) AddController(id uint8, c trans.Writer) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.controllers[id] = c
}

func (r *Device) RemoveController(id uint8) {
	/*
	 * Start of write lock
	 */
	r.mutex.Lock()
	delete(r.controllers, id)
	nCtrls := len(r.controllers)
	r.mutex.Unlock()
	/*
	 * End of write lock
	 */

	// We have no controllers?
	if nCtrls == 0 {
		// To avoid deadlock, we first unlock the mutex before calling a watcher function
		r.watcher.DeviceRemoved(r)
	}
}

func (r *Device) Descriptions() Descriptions {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.descriptions
}

func (r *Device) SetDescriptions(d Descriptions) {
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

func (r *Device) SetFeatures(f Features) {
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

func (r *Device) AddPort(num uint, p openflow.Port) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.setPort(num, p)
}

func (r *Device) UpdatePort(num uint, p openflow.Port) {
	/*
	 * Start of write lock
	 */
	r.mutex.Lock()
	port := r.ports[num]
	if port == nil {
		r.setPort(num, p)
	} else {
		port.SetValue(p)
	}
	r.mutex.Unlock()
	/*
	 * End of write lock
	 */
	if port == nil {
		return
	}

	if p.IsPortDown() || p.IsLinkDown() {
		// To avoid deadlock, we first unlock the mutex before calling a watcher function
		r.watcher.PortRemoved(port)
	}

	// TODO: Send this event to a watcher
}

func (r *Device) FlowTableID() uint8 {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.flowTableID
}

func (r *Device) SetFlowTableID(id uint8) {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.flowTableID = id
}

func (r *Device) SendMessage(msg encoding.BinaryMarshaler) error {
	// Write lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	c, ok := r.controllers[0]
	if !ok {
		return errors.New("not found main transceiver connection whose aux ID is 0")
	}

	return c.Write(msg)
}
