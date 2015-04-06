/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package device

import (
	"sync"
)

type Device struct {
	mutex    sync.Mutex
	dpid     uint64
	nBuffers uint
	nTables  uint
	//ports      map[uint]Port
	transceivers map[uint]Transceiver
}

func newDevice(dpid uint64) *Device {
	return &Device{
		dpid: dpid,
		// ports: make(map[uint]Port),
		transceivers: make(map[uint]Transceiver),
	}
}

//func (r *Device) SetPort(id uint, p Port) {
//	r.mutex.Lock()
//	defer r.mutex.Unlock()
//	r.ports[id] = p
//}
//
//func (r *Device) Port(id uint) (Port, bool) {
//	r.mutex.Lock()
//	defer r.mutex.Unlock()
//	return r.ports[id]
//}

func (r *Device) AddTransceiver(id uint, t Transceiver) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.transceivers[id] = t
}

func (r *Device) RemoveTransceiver(id uint) int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.transceivers, id)
	return len(r.transceivers)
}

func (r *Device) Transceiver(id uint) (t Transceiver, ok bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	t, ok = r.transceivers[id]
	return
}

func (r *Device) FirstTransceiver() (t Transceiver, ok bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.transceivers) == 0 {
		return nil, false
	}

	for _, t = range r.transceivers {
		break
	}

	return t, true
}
