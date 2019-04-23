/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015-2019 Samjung Data Service, Inc. All rights reserved.
 *
 *  Kitae Kim <superkkt@sds.co.kr>
 *  Donam Kim <donam.kim@sds.co.kr>
 *  Jooyoung Kang <jooyoung.kang@sds.co.kr>
 *  Changjin Choi <ccj9707@sds.co.kr>
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
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/superkkt/cherry/network"
	"github.com/superkkt/cherry/northbound/app"

	"github.com/superkkt/go-logging"
)

var (
	logger = logging.MustGetLogger("announcer")
)

// Announcer periodically broadcasts ARP announcements to update ARP cache tables on all hosts in the network.
type Announcer struct {
	app.BaseProcessor
	db   database
	once sync.Once
}

type database interface {
	RenewARPTable() error
	GetARPTable() ([]ARPTableEntry, error)
}

type ARPTableEntry struct {
	IP  net.IP
	MAC net.HardwareAddr
}

func New(db database) *Announcer {
	return &Announcer{
		db: db,
	}
}

func (r *Announcer) Init() error {
	return r.db.RenewARPTable()
}

func (r *Announcer) Name() string {
	return "Announcer"
}

func (r *Announcer) String() string {
	return fmt.Sprintf("%v", r.Name())
}

func (r *Announcer) OnDeviceUp(finder network.Finder, device *network.Device) error {
	// Make sure that there is only one broadcaster in this application.
	r.once.Do(func() {
		// Run the background broadcaster for periodic ARP announcement.
		go r.broadcaster(finder)
	})

	return r.BaseProcessor.OnDeviceUp(finder, device)
}

func (r *Announcer) broadcaster(finder network.Finder) {
	logger.Debug("executed ARP announcement broadcaster")

	backoff := newBackoff(finder)
	ticker := time.Tick(30 * time.Second)
	// Infinite loop.
	for range ticker {
		entries, err := r.db.GetARPTable()
		if err != nil {
			logger.Errorf("failed to get ARP table entries: %v", err)
			continue
		}

		for _, v := range entries {
			logger.Debugf("broadcasting an ARP announcement for a host: IP=%v, MAC=%v", v.IP, v.MAC)

			if err := backoff.Broadcast(v.IP, v.MAC); err != nil {
				logger.Errorf("failed to broadcast an ARP announcement: %v", err)
				continue
			}
			// Sleep to mitigate the peak latency of processing PACKET_INs.
			time.Sleep(time.Duration(10+rand.Intn(100)) * time.Millisecond)
		}
	}
}
