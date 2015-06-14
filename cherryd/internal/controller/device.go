/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package controller

import (
	"encoding"
	"errors"
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
	finder       Finder
	controllers  map[uint]*Controller
	descriptions Descriptions
	features     Features
	ports        map[uint]*Port
	flowTableID  uint8 // Table IDs that we install flows
	auxID        uint
}

func NewDevice(id string, log log.Logger, w Watcher, f Finder) *Device {
	return &Device{
		id:          id,
		log:         log,
		watcher:     w,
		finder:      f,
		controllers: make(map[uint]*Controller),
		ports:       make(map[uint]*Port),
	}
}

func (r *Device) ID() string {
	return r.id
}

func (r *Device) addConn(c net.Conn) {
	if c == nil {
		panic("nil connection")
	}

	/*
	 * Start of write lock
	 */
	r.mutex.Lock()
	ctr := NewController(r, c, r.log, r.watcher, r.finder)
	r.controllers[r.auxID] = ctr
	id := r.auxID
	r.auxID++
	r.mutex.Unlock()
	/*
	 * End of write lock
	 */

	go func() {
		r.log.Debug("Starting a controller..")
		ctr.Run()
		r.log.Debug("Controller is disconnected")
		r.disconnected(id)
	}()
}

func (r *Device) disconnected(id uint) {
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
	r.log.Debug("updatePort() is called..")

	/*
	 * Start of write lock
	 */
	r.mutex.Lock()
	port := r.ports[num]
	if port == nil {
		r.log.Debug("not found a port")
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

	r.log.Debug("Mutex is unlocked in updatePort()..")

	if p.IsPortDown() || p.IsLinkDown() {
		r.log.Debug("Calling PortRemoved()..")
		// To avoid deadlock, we first unlock the mutex before calling a watcher function
		r.watcher.PortRemoved(port)
		r.log.Debug("Calling PortRemoved() is done..")
	}

	// TODO: Send this event to a watcher
}

func (r *Device) FlowTableID() uint8 {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.flowTableID
}

func (r *Device) setFlowTableID(id uint8) {
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

	return c.SendMessage(msg)
}
