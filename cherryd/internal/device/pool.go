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

// TODO: This pool should be move to Zookeeper

var Pool *DevicePool

type DevicePool struct {
	mutex sync.Mutex
	// map(key=DPID, value=map(key=AuxID, value=Manager))
	pool map[uint64]map[uint8]*Manager
}

func init() {
	Pool = &DevicePool{pool: make(map[uint64]map[uint8]*Manager)}
}

func (r *DevicePool) add(dpid uint64, auxID uint8, manager *Manager) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	m, ok := r.pool[dpid]
	if ok {
		m[auxID] = manager
		return
	}
	m = make(map[uint8]*Manager)
	m[auxID] = manager
	r.pool[dpid] = m
}

func (r *DevicePool) remove(dpid uint64, auxID uint8) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	m, ok := r.pool[dpid]
	if !ok {
		return
	}
	delete(m, auxID)
	if len(m) == 0 {
		delete(r.pool, dpid)
	}
}

func (r *DevicePool) Search(dpid uint64) map[uint8]*Manager {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.pool[dpid]
}
