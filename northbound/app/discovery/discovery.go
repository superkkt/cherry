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

package discovery

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/superkkt/cherry/network"
	"github.com/superkkt/cherry/northbound/app"
	"github.com/superkkt/cherry/protocol"

	"github.com/superkkt/go-logging"
)

var (
	logger = logging.MustGetLogger("discovery")

	// A locally administered MAC address (https://en.wikipedia.org/wiki/MAC_address#Universal_vs._local).
	myMAC = net.HardwareAddr([]byte{0x06, 0xff, 0x82, 0x87, 0x29, 0x34})
)

const (
	ProbeInterval = 30 * time.Second
)

type processor struct {
	app.BaseProcessor
	db Database

	mutex     sync.Mutex
	canceller map[string]context.CancelFunc // Key = Device ID.
}

type Database interface {
	// GetUndiscoveredHosts returns IP addresses whose physical location is still
	// undiscovered or staled more than expiration.
	GetUndiscoveredHosts(expiration time.Duration) ([]net.IPNet, error)

	// UpdateHostLocation updates the physical location of a host, whose MAC and IP
	// addresses are matched with mac and ip, to the port identified by swDPID and
	// portNum. updated will be true if its location has been actually updated.
	UpdateHostLocation(mac net.HardwareAddr, ip net.IP, swDPID uint64, portNum uint16) (updated bool, err error)

	// ResetHostLocationsByPort sets NULL to the host locations that belong to the
	// port specified by swDPID and portNum.
	ResetHostLocationsByPort(swDPID uint64, portNum uint16) error

	// ResetHostLocationsByDevice sets NULL to the host locations that belong to the
	// device specified by swDPID.
	ResetHostLocationsByDevice(swDPID uint64) error
}

func New(db Database) app.Processor {
	return &processor{
		db:        db,
		canceller: make(map[string]context.CancelFunc),
	}
}

func (r *processor) Name() string {
	return "Discovery"
}

func (r *processor) String() string {
	return fmt.Sprintf("%v", r.Name())
}

func (r *processor) OnDeviceUp(finder network.Finder, device *network.Device) error {
	// Make sure that there is only one ARP sender for a device.
	r.stopARPSender(device.ID())
	r.runARPSender(device)

	// Propagate this event to the next processors.
	return r.BaseProcessor.OnDeviceUp(finder, device)
}

func (r *processor) runARPSender(device *network.Device) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Infinite loop.
		for {
			select {
			case <-ctx.Done():
				logger.Debugf("terminating the ARP sender: deviceID=%v", device.ID())
				return
			default:
			}

			if err := r.sendARPProbes(device); err != nil {
				logger.Errorf("failed to send ARP probes: %v", err)
				// Ignore this error and keep go on.
			}

			// This sleep delay should be shorter than ProbeInterval.
			time.Sleep(1500 * time.Millisecond)
		}
	}()
	r.canceller[device.ID()] = cancel
}

func (r *processor) sendARPProbes(device *network.Device) error {
	if device.IsClosed() {
		return fmt.Errorf("already closed deivce: id=%v", device.ID())
	}

	hosts, err := r.db.GetUndiscoveredHosts(ProbeInterval)
	if err != nil {
		return err
	}
	for _, addr := range hosts {
		reserved, err := network.ReservedIP(addr)
		if err != nil {
			return err
		}

		if err := device.SendARPProbe(myMAC, reserved, addr.IP); err != nil {
			return err
		}
		logger.Debugf("sent an ARP probe for %v on %v", addr.IP, device.ID())
		// Sleep to mitigate the peak latency of processing PACKET_INs.
		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

func (r *processor) stopARPSender(deviceID string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	cancel, ok := r.canceller[deviceID]
	if !ok {
		return
	}
	cancel()
	delete(r.canceller, deviceID)
}

func (r *processor) OnPacketIn(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet) error {
	// ARP?
	if eth.Type != 0x0806 {
		return r.BaseProcessor.OnPacketIn(finder, ingress, eth)
	}

	arp := new(protocol.ARP)
	if err := arp.UnmarshalBinary(eth.Payload); err != nil {
		return err
	}
	logger.Debugf("received ARP packet: %v", arp)

	switch arp.Operation {
	case 1:
		return r.processARPRequest(finder, ingress, eth, arp)
	case 2:
		return r.processARPReply(finder, ingress, eth, arp)
	default:
		logger.Debugf("dropping the ARP packet that has invalid operaion code: %v", arp)
		// Drop this packet. Do not pass it to the next processors.
		return nil
	}
}

func (r *processor) processARPRequest(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet, arp *protocol.ARP) error {
	// Our ARP probe?
	if bytes.Equal(arp.SHA, myMAC) {
		// Drop this packet! This packet should not be propagated among switches.
		logger.Debugf("dropping our ARP probe that was propagated via an edge among switches: deviceID=%v", ingress.Device().ID())
		return nil
	} else {
		// Propagate this ARP request, wich is raised from a host, to the next processors.
		return r.BaseProcessor.OnPacketIn(finder, ingress, eth)
	}
}

func (r *processor) processARPReply(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet, arp *protocol.ARP) error {
	// The target (not source!) hardware address of the ARP reply packet (or the destination MAC address of the ethernet
	// frame) should be equal to the myMAC address if it is a counterpart for our ARP probe.
	if bytes.Equal(arp.THA, myMAC) == false && bytes.Equal(eth.DstMAC, myMAC) == false {
		logger.Debugf("unexpected ARP reply: %v", arp)
		// Drop this packet. Do not pass it to the next processors.
		return nil
	}
	if finder.IsEdge(ingress) {
		logger.Debugf("dropping ARP reply received from an edge among switches: ingress=%v, arp=%v", ingress.ID(), arp)
		// Drop this packet. Do not pass it to the next processors.
		return nil
	}

	// This ARP reply packet has been processed. Do not pass it to the next processors.
	return r.macLearning(finder, ingress, arp)
}

func (r *processor) macLearning(finder network.Finder, ingress *network.Port, arp *protocol.ARP) error {
	swDPID, err := strconv.ParseUint(ingress.Device().ID(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid device ID: %v", ingress.Device().ID())
	}

	// Update the host location in the database if SHA and SPA are matched.
	updated, err := r.db.UpdateHostLocation(arp.SHA, arp.SPA, swDPID, uint16(ingress.Number()))
	if err != nil {
		return err
	}
	// Remove installed flows for this host if the location has been changed.
	if updated {
		logger.Infof("update host location: IP=%v, MAC=%v, deviceID=%v, portNum=%v", arp.SPA, arp.SHA, swDPID, ingress.Number())
		// Remove flows from all devices.
		for _, device := range finder.Devices() {
			if err := device.RemoveFlowByMAC(arp.SHA); err != nil {
				logger.Errorf("failed to remove flows from %v: %v", device.ID(), err)
				continue
			}
			logger.Debugf("removed flows whose destination MAC address is %v on %v", arp.SHA, device.ID())
		}
	} else {
		logger.Debugf("skip to update host location: unknown host or no location change: IP=%v, MAC=%v, deviceID=%v, portNum=%v", arp.SPA, arp.SHA, swDPID, ingress.Number())
	}

	return nil
}

func (r *processor) OnPortDown(finder network.Finder, port *network.Port) error {
	swDPID, err := strconv.ParseUint(port.Device().ID(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid device ID: %v", port.Device().ID())
	}

	// Set NULLs to the host locations that associated with this port so that the
	// packets heading to these hosts will be broadcasted until we discover it again.
	if err := r.db.ResetHostLocationsByPort(swDPID, uint16(port.Number())); err != nil {
		return err
	}

	// Propagate this event to the next processors.
	return r.BaseProcessor.OnPortDown(finder, port)
}

func (r *processor) OnDeviceDown(finder network.Finder, device *network.Device) error {
	// Stop the ARP request sender.
	r.stopARPSender(device.ID())

	swDPID, err := strconv.ParseUint(device.ID(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid device ID: %v", device.ID())
	}

	// Set NULLs to the host locations that belong to this device so that the packets
	// heading to these hosts will be broadcasted until we discover them again.
	if err := r.db.ResetHostLocationsByDevice(swDPID); err != nil {
		return err
	}

	// Propagate this event to the next processors.
	return r.BaseProcessor.OnDeviceDown(finder, device)
}

func (r *processor) OnTopologyChange(finder network.Finder) error {
	for _, device := range finder.Devices() {
		swDPID, err := strconv.ParseUint(device.ID(), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid device ID: %v", device.ID())
		}

		for _, port := range device.Ports() {
			if finder.IsEdge(port) == false {
				continue
			}

			if err := r.db.ResetHostLocationsByPort(swDPID, uint16(port.Number())); err != nil {
				logger.Errorf("failed to reset host locations by port: DPID=%v, PortNum=%v, err=%v", swDPID, port.Number(), err)
				continue
			}
			logger.Debugf("reset host locations that belong to the edge port: DPID=%v, PortNum=%v", swDPID, port.Number())
		}
	}

	// Propagate this event to the next processors.
	return r.BaseProcessor.OnTopologyChange(finder)
}
