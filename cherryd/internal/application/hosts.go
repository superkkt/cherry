/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package application

import (
	"net"
	"sync"
)

var hostDB *Hosts

func init() {
	hostDB = NewHosts()
	// FIXME: Read router IP addresses from DB
	hostDB.Add(net.IPv4(223, 130, 122, 1), virtualMAC)
	hostDB.Add(net.IPv4(223, 130, 123, 1), virtualMAC)
	hostDB.Add(net.IPv4(223, 130, 124, 1), virtualMAC)
	hostDB.Add(net.IPv4(223, 130, 125, 1), virtualMAC)
	hostDB.Add(net.IPv4(10, 0, 0, 254), virtualMAC)
	hostDB.Add(net.IPv4(10, 0, 0, 1), net.HardwareAddr([]byte{0, 0, 0, 0, 0, 1}))
	hostDB.Add(net.IPv4(10, 0, 0, 2), net.HardwareAddr([]byte{0, 0, 0, 0, 0, 2}))
}

type Hosts struct {
	mutex sync.Mutex
	hosts map[string]net.HardwareAddr // Key is the IP address
}

func NewHosts() *Hosts {
	return &Hosts{
		hosts: make(map[string]net.HardwareAddr),
	}
}

func (r *Hosts) Add(ip net.IP, mac net.HardwareAddr) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if ip == nil || mac == nil {
		panic("nil IP or MAC address")
	}
	r.hosts[ip.String()] = mac
}

func (r *Hosts) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.hosts = make(map[string]net.HardwareAddr)
}

func (r *Hosts) MAC(ip net.IP) (mac net.HardwareAddr, ok bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if ip == nil {
		panic("nil IP address")
	}

	mac, ok = r.hosts[ip.String()]
	return
}
