/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package device

import (
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"sync"
)

type Device struct {
	mutex        sync.Mutex
	DPID         uint64
	NumBuffers   uint
	NumTables    uint
	ports        map[uint]openflow.Port
	transceivers map[uint]Transceiver
	Manufacturer string
	Hardware     string
	Software     string
	Serial       string
	Description  string
}

func newDevice(dpid uint64) *Device {
	return &Device{
		DPID:         dpid,
		ports:        make(map[uint]openflow.Port),
		transceivers: make(map[uint]Transceiver),
	}
}

func (r *Device) setPort(id uint, p openflow.Port) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.ports[id] = p
}

func (r *Device) Port(id uint) (p openflow.Port, ok bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	p, ok = r.ports[id]
	return
}

func (r *Device) addTransceiver(id uint, t Transceiver) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.transceivers[id] = t
}

// removeTransceiver returns the number of remaining transceivers after removing.
func (r *Device) removeTransceiver(id uint) int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.transceivers, id)
	return len(r.transceivers)
}

func (r *Device) getTransceiver() Transceiver {
	for _, v := range r.transceivers {
		// Return the first transceiver
		return v
	}

	panic("empty transceiver in a device!")
}

func (r *Device) SetBarrier() error {
	t := r.getTransceiver()
	return t.sendBarrierRequest()
}

func (r *Device) NewMatch() openflow.Match {
	t := r.getTransceiver()
	return t.newMatch()
}

func (r *Device) NewAction() openflow.Action {
	t := r.getTransceiver()
	return t.newAction()
}

func (r *Device) InstallFlowRule(conf FlowModConfig) error {
	t := r.getTransceiver()
	return t.addFlowMod(conf)
}

// TODO: Add exposed functions to provide OpenFlow funtionality to plugins.
