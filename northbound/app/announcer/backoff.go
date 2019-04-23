/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service, Inc. All rights reserved.
 * Kitae Kim <superkkt@sds.co.kr>
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along
 * with this program; if not, write to the Free Software Foundation, Inc.,
 * 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
 */

package announcer

import (
	"bytes"
	"math"
	"net"
	"sync"
	"time"

	"github.com/superkkt/cherry/network"

	lru "github.com/hashicorp/golang-lru"
)

type backoff struct {
	finder network.Finder

	mutex sync.Mutex
	cache *lru.Cache
}

func newBackoff(finder network.Finder) *backoff {
	if finder == nil {
		panic("nil finder parameter")
	}
	cache, err := lru.New(16384)
	if err != nil {
		panic(err)
	}

	return &backoff{
		finder: finder,
		cache:  cache,
	}
}

func (r *backoff) Broadcast(ip net.IP, mac net.HardwareAddr) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.getAnnouncer(ip, mac).Broadcast()
}

func (r *backoff) getAnnouncer(ip net.IP, mac net.HardwareAddr) *announcer {
	var result *announcer

	v, ok := r.cache.Get(ip.String())
	if !ok || bytes.Equal(v.(*announcer).MAC, mac) == false {
		result = &announcer{
			Finder: r.finder,
			IP:     ip,
			MAC:    mac,
		}
		r.cache.Add(ip.String(), result)
	} else {
		result = v.(*announcer)
	}

	return result
}

type announcer struct {
	Finder    network.Finder
	IP        net.IP
	MAC       net.HardwareAddr
	count     uint64
	timestamp time.Time
}

func (r *announcer) Broadcast() error {
	delay := r.calculateDelay()
	if time.Now().Sub(r.timestamp) < delay {
		// Do nothing. We need more time.
		logger.Debugf("skip to broadcast an ARP announcement for %v until %v", r.IP, r.timestamp.Add(delay))
		return nil
	}

	broadcasted := false
	for _, device := range r.Finder.Devices() {
		if err := device.SendARPAnnouncement(r.IP, r.MAC); err != nil {
			return err
		}
		logger.Debugf("sent an ARP announcement: DPID=%v, IP=%v, MAC=%v", device.ID(), r.IP, r.MAC)
		broadcasted = true
	}
	if broadcasted {
		r.count++
		r.timestamp = time.Now()
	}

	return nil
}

const maxDelay = 1 * time.Hour

func (r *announcer) calculateDelay() time.Duration {
	// Overflow?
	if float64(r.count) > math.Log2(maxDelay.Seconds()) {
		return maxDelay
	}

	// Exponential delay.
	return time.Duration(math.Pow(2, float64(r.count))) * time.Second
}
