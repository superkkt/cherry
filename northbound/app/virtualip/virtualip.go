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

package virtualip

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/superkkt/cherry/network"
	"github.com/superkkt/cherry/northbound/app"

	"github.com/superkkt/go-logging"
)

var (
	logger = logging.MustGetLogger("virtualip")
)

// NOTE: This VirtualIP module should be executed before the Discovery module.
type VirtualIP struct {
	app.BaseProcessor
	db   database
	once sync.Once
}

type database interface {
	ToggleDeviceVIP(swDPID uint64) ([]Address, error)
	TogglePortVIP(swDPID uint64, portNum uint16) ([]Address, error)
	GetActivatedVIPs() (vips []Address, err error)
}

type Address struct {
	IP  net.IP
	MAC net.HardwareAddr
}

func New(db database) *VirtualIP {
	return &VirtualIP{
		db: db,
	}
}

func (r *VirtualIP) Init() error {
	return nil
}

func (r *VirtualIP) broadcaster(finder network.Finder) {
	logger.Debug("executed ARP announcement broadcaster")

	ticker := time.Tick(30 * time.Second)
	// Infinite loop.
	for range ticker {
		vips, err := r.db.GetActivatedVIPs()
		if err != nil {
			logger.Errorf("failed to get activated VIPs: %v", err)
			continue
		}

		broadcastARPAnnouncement(finder, vips, false)
	}
}

func (r *VirtualIP) Name() string {
	return "VirtualIP"
}

func (r *VirtualIP) String() string {
	return fmt.Sprintf("%v", r.Name())
}

func (r *VirtualIP) OnDeviceUp(finder network.Finder, device *network.Device) error {
	r.once.Do(func() {
		// Run the background broadcaster for periodic ARP announcement.
		go r.broadcaster(finder)
	})

	return r.BaseProcessor.OnDeviceUp(finder, device)
}

func (r *VirtualIP) OnPortDown(finder network.Finder, port *network.Port) error {
	logger.Debugf("port down! checking VIPs that belong to the port... (DPID=%v, Port=%v)", port.Device().ID(), port.Number())

	dpid, err := strconv.ParseUint(port.Device().ID(), 10, 64)
	if err != nil {
		logger.Errorf("invalid switch DPID: %v", port.Device().ID())
		return r.BaseProcessor.OnPortDown(finder, port)
	}
	vips, err := r.db.TogglePortVIP(dpid, uint16(port.Number()))
	if err != nil {
		logger.Errorf("failed to toggle VIP hosts: %v", err)
		return r.BaseProcessor.OnPortDown(finder, port)
	}
	broadcastARPAnnouncement(finder, vips, true)

	return r.BaseProcessor.OnPortDown(finder, port)
}

func (r *VirtualIP) OnDeviceDown(finder network.Finder, device *network.Device) error {
	logger.Debugf("device down! checking VIPs that belong to the device... (DPID=%v)", device.ID())

	dpid, err := strconv.ParseUint(device.ID(), 10, 64)
	if err != nil {
		logger.Errorf("invalid switch DPID: %v", device.ID())
		return r.BaseProcessor.OnDeviceDown(finder, device)
	}
	vips, err := r.db.ToggleDeviceVIP(dpid)
	if err != nil {
		logger.Errorf("failed to toggle VIP hosts: %v", err)
		return r.BaseProcessor.OnDeviceDown(finder, device)
	}
	broadcastARPAnnouncement(finder, vips, true)

	return r.BaseProcessor.OnDeviceDown(finder, device)
}

func broadcastARPAnnouncement(finder network.Finder, vips []Address, toggled bool) {
	for _, v := range vips {
		for _, d := range finder.Devices() {
			if err := d.SendARPAnnouncement(v.IP, v.MAC); err != nil {
				logger.Errorf("failed to broadcast ARP announcement: %v", err)
				continue
			}
			logger.Debugf("sent an ARP announcement for VIP: DPID=%v, IP=%v, MAC=%v", d.ID(), v.IP, v.MAC)
		}

		if toggled {
			logger.Warningf("VIP toggled: IP=%v, MAC=%v", v.IP, v.MAC)
		}
	}
}
