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
	MAC(ip net.IP) (mac net.HardwareAddr, ok bool, err error)
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

	r.log.Debug(fmt.Sprintf("ProxyARP: received ARP packet.. ingress=%v, srcEthMAC=%v, dstEthMAC=%v", ingress.ID(), eth.SrcMAC, eth.DstMAC))

	arp := new(protocol.ARP)
	if err := arp.UnmarshalBinary(eth.Payload); err != nil {
		return err
	}
	// Drop ARP announcement
	if isARPAnnouncement(arp) {
		// We don't allow a host sends ARP announcement to the network. This controller only can send it,
		// and we will flood the announcement to all switch devices using PACKET_OUT  when we need it.
		r.log.Info(fmt.Sprintf("ProxyARP: drop ARP announcements.. ingress=%v (%v)", ingress.ID(), arp))
		return nil
	}
	// ARP request?
	if arp.Operation != 1 {
		// Drop all ARP packets whose type is not a reqeust.
		r.log.Info(fmt.Sprintf("ProxyARP: drop ARP packet whose type is not a request.. ingress=%v (%v)", ingress.ID(), arp))
		return nil
	}
	mac, ok, err := r.db.MAC(arp.TPA)
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
	r.log.Debug(fmt.Sprintf("ProxyARP: sending ARP reply to %v..", ingress.ID()))
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
	sameProtoAddr := request.SPA.Equal(request.TPA)
	sameHWAddr := bytes.Compare(request.SHA, request.THA) == 0
	zeroTarget := bytes.Compare(request.THA, []byte{0, 0, 0, 0, 0, 0}) == 0
	broadcastTarget := bytes.Compare(request.THA, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}) == 0
	if sameProtoAddr && (zeroTarget || broadcastTarget || sameHWAddr) {
		return true
	}

	return false
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
