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
	"net"
	"strconv"

	"github.com/superkkt/cherry/network"
	"github.com/superkkt/cherry/northbound/app"
	"github.com/superkkt/cherry/openflow"
	"github.com/superkkt/cherry/protocol"

	"github.com/pkg/errors"
	"github.com/superkkt/go-logging"
)

var (
	logger = logging.MustGetLogger("proxyarp")
)

type ProxyARP struct {
	app.BaseProcessor
	db database
}

type database interface {
	MAC(ip net.IP) (mac net.HardwareAddr, ok bool, err error)
	ToggleDeviceVIP(swDPID uint64) ([]VIP, error)
	TogglePortVIP(swDPID uint64, portNum uint16) ([]VIP, error)
}

func New(db database) *ProxyARP {
	return &ProxyARP{
		db: db,
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

	logger.Debugf("received ARP packet.. ingress=%v, srcEthMAC=%v, dstEthMAC=%v", ingress.ID(), eth.SrcMAC, eth.DstMAC)

	arp := new(protocol.ARP)
	if err := arp.UnmarshalBinary(eth.Payload); err != nil {
		return err
	}
	// Drop ARP announcement
	if isARPAnnouncement(arp) {
		// We don't allow a host sends ARP announcement to the network. This controller only can send it,
		// and we will flood the announcement to all switch devices using PACKET_OUT  when we need it.
		logger.Infof("drop ARP announcements.. ingress=%v (%v)", ingress.ID(), arp)
		return nil
	}
	// ARP request?
	if arp.Operation != 1 {
		// Drop all ARP packets whose type is not a reqeust.
		logger.Infof("drop ARP packet whose type is not a request.. ingress=%v (%v)", ingress.ID(), arp)
		return nil
	}

	mac, ok, err := r.db.MAC(arp.TPA)
	if err != nil {
		return errors.Wrap(&proxyarpErr{temporary: true, err: err}, "failed to query MAC")
	}
	if !ok {
		logger.Debugf("drop the ARP request for unknown host (%v)", arp.TPA)
		// Unknown hosts. Drop the packet.
		return nil
	}
	logger.Debugf("ARP request for %v (%v)", arp.TPA, mac)

	reply, err := makeARPReply(arp, mac)
	if err != nil {
		return err
	}
	logger.Debugf("sending ARP reply to %v..", ingress.ID())

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

type VIP struct {
	Address net.IP
	MAC     net.HardwareAddr
}

func (r *ProxyARP) OnPortDown(finder network.Finder, port *network.Port) error {
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
	r.broadcastARPAnnouncement(finder, vips)

	return r.BaseProcessor.OnPortDown(finder, port)
}

func (r *ProxyARP) OnDeviceDown(finder network.Finder, device *network.Device) error {
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
	r.broadcastARPAnnouncement(finder, vips)

	return r.BaseProcessor.OnDeviceDown(finder, device)
}

func (r *ProxyARP) broadcastARPAnnouncement(finder network.Finder, vips []VIP) {
	for _, v := range vips {
		for _, d := range finder.Devices() {
			if err := d.SendARPAnnouncement(v.Address, v.MAC); err != nil {
				logger.Errorf("failed to broadcast ARP announcement: %v", err)
				continue
			}
		}
		logger.Warningf("VIP toggled: IP=%v, MAC=%v", v.Address, v.MAC)
	}
}
