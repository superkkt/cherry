/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package device

import (
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"sync"
)

type port struct {
	value openflow.Port
	link  *Edge
}

type Device struct {
	mutex        sync.Mutex
	DPID         uint64
	NumBuffers   uint
	NumTables    uint
	ports        map[uint]*port
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
		ports:        make(map[uint]*port),
		transceivers: make(map[uint]Transceiver),
	}
}

func (r Device) ID() string {
	return fmt.Sprintf("%v", r.DPID)
}

func (r *Device) setPort(id uint, p openflow.Port) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.ports[id] = &port{value: p}
}

func (r *Device) Port(id uint) (*port, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	p, ok := r.ports[id]
	return p, ok
}

func (r *Device) Ports() []*port {
	ports := make([]*port, 0)
	for _, v := range r.ports {
		ports = append(ports, v)
	}

	return ports
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

func (r *Device) PacketOut(inport openflow.InPort, action openflow.Action, data []byte) error {
	t := r.getTransceiver()
	return t.packetOut(inport, action, data)
}

func (r *Device) Flood(inPort openflow.InPort, data []byte) error {
	t := r.getTransceiver()
	return t.flood(inPort, data)
}

// TODO: Add exposed functions to provide OpenFlow funtionality to plugins.
