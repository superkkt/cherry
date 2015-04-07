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

var Pool *DevicePool

type DevicePool struct {
	mutex sync.Mutex
	pool  map[uint64]*Device
}

func init() {
	Pool = &DevicePool{
		pool: make(map[uint64]*Device),
	}
}

func (r *DevicePool) add(dpid uint64, d *Device) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.pool[dpid] = d
}

func (r *DevicePool) remove(dpid uint64) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.pool, dpid)
}

func (r *DevicePool) Get(dpid uint64) *Device {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.pool[dpid]
}

func findDevice(dpid uint64) *Device {
	v := Pool.Get(dpid)
	if v != nil {
		return v
	}

	v = newDevice(dpid)
	Pool.add(dpid, v)

	return v
}
