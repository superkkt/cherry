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

package proxyarp

import (
	"bytes"
	"fmt"
	"github.com/dlintw/goconf"
	"github.com/superkkt/cherry/cherryd/log"
	"github.com/superkkt/cherry/cherryd/network"
	"github.com/superkkt/cherry/cherryd/northbound/app"
	"github.com/superkkt/cherry/cherryd/openflow"
	"github.com/superkkt/cherry/cherryd/protocol"
	"net"
)

type ProxyARP struct {
	app.BaseProcessor
	conf *goconf.ConfigFile
	log  log.Logger
	db   database
}

type database interface {
	FindMAC(ip net.IP) (mac net.HardwareAddr, ok bool, err error)
}

func New(conf *goconf.ConfigFile, log log.Logger, db database) *ProxyARP {
	return &ProxyARP{
		conf: conf,
		log:  log,
		db:   db,
	}
}

func (r *ProxyARP) Init() error {
	return nil
}

func (r *ProxyARP) Name() string {
	return "ProxyARP"
}

func (r *ProxyARP) OnPacketIn(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet) error {
	// ARP?
	if eth.Type != 0x0806 {
		return r.BaseProcessor.OnPacketIn(finder, ingress, eth)
	}

	r.log.Debug("ProxyARP: received ARP packet..")

	arp := new(protocol.ARP)
	if err := arp.UnmarshalBinary(eth.Payload); err != nil {
		return err
	}
	// ARP request?
	if arp.Operation != 1 {
		r.log.Debug("ProxyARP: drop ARP packet whose type is not a request")
		// Drop all ARP packets if their type is not a reqeust.
		return nil
	}
	// Pass ARP announcements packets if it has valid source IP & MAC addresses
	if isARPAnnouncement(arp) {
		r.log.Debug("ProxyARP: received ARP announcements..")
		valid, err := r.isValidARPAnnouncement(arp)
		if err != nil {
			return err
		}
		if !valid {
			// Drop suspicious announcement packet
			r.log.Info(fmt.Sprintf("ProxyARP: drop suspicious ARP announcement from %v to %v", eth.SrcMAC.String(), eth.DstMAC.String()))
			return nil
		}
		r.log.Debug("ProxyARP: pass valid ARP announcements into the network")
		// Pass valid ARP announcements to the network
		return r.BaseProcessor.OnPacketIn(finder, ingress, eth)
	}
	mac, ok, err := r.db.FindMAC(arp.TPA)
	if err != nil {
		return err
	}
	if !ok {
		r.log.Debug(fmt.Sprintf("ProxyARP: drop the ARP request for unknown host (%v)", arp.TPA))
		// Unknown hosts. Drop the packet.
		return nil
	}
	r.log.Debug(fmt.Sprintf("ProxyARP: ARP request for %v (%v)", arp.TPA, mac))

	reply, err := makeARPReply(arp, mac)
	if err != nil {
		return err
	}
	r.log.Debug("ProxyARP: sending ARP reply..")
	return sendARPReply(ingress, reply)
}

func sendARPReply(ingress *network.Port, packet []byte) error {
	f := ingress.Device().Factory()

	inPort := openflow.NewInPort()
	inPort.SetController()

	outPort := openflow.NewOutPort()
	outPort.SetValue(ingress.Number())

	action, err := f.NewAction()
	if err != nil {
		return err
	}
	action.SetOutPort(outPort)

	out, err := f.NewPacketOut()
	if err != nil {
		return err
	}
	out.SetInPort(inPort)
	out.SetAction(action)
	out.SetData(packet)

	return ingress.Device().SendMessage(out)
}

func isARPAnnouncement(request *protocol.ARP) bool {
	sameAddr := request.SPA.Equal(request.TPA)
	zeroTarget := bytes.Compare(request.THA, []byte{0, 0, 0, 0, 0, 0}) == 0
	if !sameAddr || !zeroTarget {
		return false
	}

	return true
}

func (r *ProxyARP) isValidARPAnnouncement(request *protocol.ARP) (bool, error) {
	// Trusted MAC address?
	mac, ok, err := r.db.FindMAC(request.SPA)
	if err != nil {
		return false, err
	}
	if !ok || bytes.Compare(mac, request.SHA) != 0 {
		// Suspicious announcemens
		return false, nil
	}

	return true, nil
}

func makeARPReply(request *protocol.ARP, mac net.HardwareAddr) ([]byte, error) {
	v := protocol.NewARPReply(mac, request.SHA, request.TPA, request.SPA)
	reply, err := v.MarshalBinary()
	if err != nil {
		return nil, err
	}
	eth := protocol.Ethernet{
		SrcMAC:  mac,
		DstMAC:  request.SHA,
		Type:    0x0806,
		Payload: reply,
	}

	return eth.MarshalBinary()
}

func (r *ProxyARP) String() string {
	return fmt.Sprintf("%v", r.Name())
}
