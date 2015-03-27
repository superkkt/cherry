/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package device

// Key: DPID, Value: Device Manager
var Pool map[uint64]*Manager

func init() {
	Pool = make(map[uint64]*Manager)
}

func add(dpid uint64, device *Manager) {
	Pool[dpid] = device
}

func remove(dpid uint64) {
	delete(Pool, dpid)
}

func Search(dpid uint64) *Manager {
	return Pool[dpid]
}
