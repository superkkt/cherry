/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
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
	"github.com/superkkt/cherry/cherryd/internal/log"
	"github.com/superkkt/cherry/cherryd/internal/network"
	"github.com/superkkt/cherry/cherryd/internal/northbound/app"
	"github.com/superkkt/cherry/cherryd/openflow"
	"github.com/superkkt/cherry/cherryd/protocol"
	"github.com/dlintw/goconf"
	"net"
)

type ProxyARP struct {
	app.BaseProcessor
	conf  *goconf.ConfigFile
	log   log.Logger
	hosts *database
}

func New(conf *goconf.ConfigFile, log log.Logger) *ProxyARP {
	return &ProxyARP{
		conf: conf,
		log:  log,
	}
}

func (r *ProxyARP) Init() error {
	db, err := newDatabase(r.conf)
	if err != nil {
		return err
	}
	r.hosts = db

	return nil
}

func (r *ProxyARP) Name() string {
	return "ProxyARP"
}

func (r *ProxyARP) execNextOnPacketIn(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet) error {
	next, ok := r.Next()
	if !ok {
		return nil
	}
	return next.OnPacketIn(finder, ingress, eth)
}

func (r *ProxyARP) OnPacketIn(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet) error {
	// ARP?
	if eth.Type != 0x0806 {
		return r.execNextOnPacketIn(finder, ingress, eth)
	}

	r.log.Debug("Received ARP packet..")

	arp := new(protocol.ARP)
	if err := arp.UnmarshalBinary(eth.Payload); err != nil {
		return err
	}
	// ARP request?
	if arp.Operation != 1 {
		r.log.Debug("Drop ARP packet whose type is not a request")
		// Drop all ARP packets if their type is not a reqeust.
		return nil
	}
	// Pass ARP announcements packets if it has valid source IP & MAC addresses
	if isARPAnnouncement(arp) {
		r.log.Debug("Received ARP announcements..")
		valid, err := r.isValidARPAnnouncement(arp)
		if err != nil {
			return err
		}
		if !valid {
			// Drop suspicious announcement packet
			r.log.Info(fmt.Sprintf("ProxyARP: drop suspicious ARP announcement from %v to %v", eth.SrcMAC.String(), eth.DstMAC.String()))
			return nil
		}
		r.log.Debug("Pass valid ARP announcements to the network")
		// Pass valid ARP announcements to the network
		return r.execNextOnPacketIn(finder, ingress, eth)
	}
	mac, ok, err := r.hosts.mac(arp.TPA)
	if err != nil {
		return err
	}
	if !ok {
		r.log.Debug(fmt.Sprintf("ARP request for unknown host (%v)", arp.TPA))
		// Unknown hosts. Drop the packet.
		return nil
	}
	reply, err := makeARPReply(arp, mac)
	if err != nil {
		return err
	}

	r.log.Debug("Sending ARP reply..")
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
	mac, ok, err := r.hosts.mac(request.SPA)
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
