/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015-2019 Samjung Data Service, Inc. All rights reserved.
 *  Kitae Kim <superkkt@sds.co.kr>
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

package dhcp

import (
	"encoding/hex"
	"fmt"
	"net"

	"github.com/superkkt/cherry/network"
	"github.com/superkkt/cherry/northbound/app"
	"github.com/superkkt/cherry/openflow"
	"github.com/superkkt/cherry/protocol"

	"github.com/davecgh/go-spew/spew"
	"github.com/superkkt/go-logging"
)

var (
	logger = logging.MustGetLogger("dhcp")

	// 192.0.2.2 is one of the reserved addresses (TEST-NET-1). See RFC 5737.
	serverIP = net.IPv4(192, 0, 2, 2)
	// A locally administered MAC address (https://en.wikipedia.org/wiki/MAC_address#Universal_vs._local).
	serverMAC = net.HardwareAddr([]byte{0x06, 0xff, 0x15, 0x88, 0x67, 0x76})
)

type DHCP struct {
	app.BaseProcessor
	db Database
}

type Database interface {
	// DHCP returns a network configuration matched with the hardware address, otherwise nil.
	DHCP(net.HardwareAddr) (*NetConfig, error)
}

type NetConfig struct {
	IP      net.IP     // Host IP address.
	Mask    net.IPMask // Network mask.
	Gateway net.IP     // Gateway (router) IP address.
}

func New(db Database) *DHCP {
	return &DHCP{
		db: db,
	}
}

func (r *DHCP) Init() error {
	return nil
}

func (r *DHCP) Name() string {
	return "DHCP"
}

func (r *DHCP) String() string {
	return fmt.Sprintf("%v", r.Name())
}

func (r *DHCP) OnPacketIn(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet) error {
	if eth.Type != 0x0800 /* IPv4 */ {
		return r.BaseProcessor.OnPacketIn(finder, ingress, eth)
	}

	ip := new(protocol.IPv4)
	if err := ip.UnmarshalBinary(eth.Payload); err != nil {
		logger.Debugf("drop an invalid IPv4 packet: %v", err)
		return nil
	}
	if ip.Protocol != 0x11 /* UDP */ {
		return r.BaseProcessor.OnPacketIn(finder, ingress, eth)
	}

	udp := new(protocol.UDP)
	if err := udp.UnmarshalBinary(ip.Payload); err != nil {
		logger.Debugf("drop an invalid UDP packet: %v", err)
		return nil
	}
	// DHCP client and server ports?
	if udp.SrcPort != 68 || udp.DstPort != 67 {
		return r.BaseProcessor.OnPacketIn(finder, ingress, eth)
	}

	dhcp := new(protocol.DHCP)
	if err := dhcp.UnmarshalBinary(udp.Payload); err != nil {
		logger.Debugf("bypass an invalid DHCP packet: %v", err)
		return r.BaseProcessor.OnPacketIn(finder, ingress, eth)
	}

	logger.Debugf("processing a DHCP packet: ingress=%v, data=%v", ingress.ID(), spew.Sdump(dhcp))
	if r.processDHCPPacket(ingress, dhcp) == true {
		logger.Infof("bypass the DHCP packet: ingress=%v, data=%v", ingress.ID(), spew.Sdump(dhcp))
		return r.BaseProcessor.OnPacketIn(finder, ingress, eth)
	}

	return nil
}

func (r *DHCP) processDHCPPacket(ingress *network.Port, dhcp *protocol.DHCP) (bypass bool) {
	t, ok := dhcp.Option(0x35) // Message type.
	if ok == false || len(t.Value) < 1 {
		logger.Debugf("unknown DHCP message type: %v", spew.Sdump(dhcp))
		return true
	}

	conf, err := r.db.DHCP(dhcp.CHAddr)
	if err != nil {
		logger.Errorf("failed to query DHCP client IP address: %v", err)
		return true
	}
	if conf == nil {
		logger.Debugf("unknown DHCP MAC address: %v", dhcp.CHAddr)
		return true
	}
	if conf.Gateway.To4() == nil {
		logger.Errorf("invalid IPv4 gateway address: %v", conf.Gateway)
		return true
	}

	switch t.Value[0] {
	case 0x01: // Discover.
		if err := r.processDHCPDiscover(ingress, dhcp, conf); err != nil {
			logger.Errorf("failed to process DHCP discover packet: %v", err)
			return true
		}
	case 0x03: // Request.
		if err := r.processDHCPRequest(ingress, dhcp, conf); err != nil {
			logger.Errorf("failed to process DHCP request packet: %v", err)
			return true
		}
	default:
		logger.Debugf("unexpected DHCP message type: %v", t.Value[0])
		return true
	}

	// Consume the DHCP packet and do not forward into other apps.
	return false
}

var (
	dns          = []byte{8, 8, 8, 8, 1, 1, 1, 1, 168, 126, 63, 1} // DNS (8.8.8.8, 1.1.1.1, 168.126.63.1).
	leaseTimeSec = []byte{0x00, 0x01, 0x51, 0x80}                  // IP Address Lease Time (86400 seconds).
	broadcastMAC = net.HardwareAddr([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	broadcastIP  = net.IPv4(255, 255, 255, 255)
)

func (r *DHCP) processDHCPDiscover(ingress *network.Port, disc *protocol.DHCP, conf *NetConfig) error {
	dhcp := &protocol.DHCP{
		Op:      protocol.DHCPOpcodeReply,
		Hops:    0,
		XID:     disc.XID,
		Elapsed: 0,
		Flags:   0,
		CIAddr:  net.IPv4zero,
		YIAddr:  conf.IP,
		SIAddr:  net.IPv4zero,
		GIAddr:  net.IPv4zero,
		CHAddr:  disc.CHAddr,
		SName:   "",
		File:    "",
		Options: []protocol.DHCPOption{
			{Code: 53, Value: []byte{2}},         // Message Type (DHCPOFFER).
			{Code: 1, Value: conf.Mask},          // Subnet Mask.
			{Code: 3, Value: conf.Gateway.To4()}, // Router.
			{Code: 6, Value: dns},                // DNS.
			{Code: 51, Value: leaseTimeSec},      // IP Address Lease Time.
			{Code: 54, Value: serverIP.To4()},    // Server Identifier.
		},
	}

	reply, err := buildReply(broadcastMAC, broadcastIP, dhcp)
	if err != nil {
		return err
	}
	logger.Debugf("sending a DHCPOFFER: %v (packet=%v)", spew.Sdump(dhcp), hex.EncodeToString(reply))

	return sendReply(ingress, reply)
}

func (r *DHCP) processDHCPRequest(ingress *network.Port, req *protocol.DHCP, conf *NetConfig) error {
	dhcp := &protocol.DHCP{
		Op:      protocol.DHCPOpcodeReply,
		Hops:    0,
		XID:     req.XID,
		Elapsed: 0,
		Flags:   0,
		CIAddr:  req.CIAddr,
		YIAddr:  conf.IP,
		SIAddr:  net.IPv4zero,
		GIAddr:  net.IPv4zero,
		CHAddr:  req.CHAddr,
		SName:   "",
		File:    "",
		Options: []protocol.DHCPOption{
			{Code: 53, Value: []byte{5}},         // Message Type (DHCPACK).
			{Code: 1, Value: conf.Mask},          // Subnet Mask.
			{Code: 3, Value: conf.Gateway.To4()}, // Router.
			{Code: 6, Value: dns},                // DNS.
			{Code: 51, Value: leaseTimeSec},      // IP Address Lease Time.
			{Code: 54, Value: serverIP.To4()},    // Server Identifier.
		},
	}

	dstMAC := broadcastMAC
	dstIP := broadcastIP
	// Renewal request?
	if req.CIAddr.Equal(net.IPv4zero) == false {
		dstMAC = req.CHAddr
		dstIP = req.CIAddr
	}

	reply, err := buildReply(dstMAC, dstIP, dhcp)
	if err != nil {
		return err
	}
	logger.Debugf("sending a DHCPACK: %v (packet=%v)", spew.Sdump(dhcp), hex.EncodeToString(reply))

	return sendReply(ingress, reply)
}

func buildReply(dstMAC net.HardwareAddr, dstIP net.IP, dhcp *protocol.DHCP) ([]byte, error) {
	payload, err := dhcp.MarshalBinary()
	if err != nil {
		return nil, err
	}

	udp := &protocol.UDP{
		SrcPort: 67,
		DstPort: 68,
		Length:  uint16(len(payload)),
		Payload: payload,
	}
	udp.SetPseudoHeader(serverIP, dstIP)
	datagram, err := udp.MarshalBinary()
	if err != nil {
		return nil, err
	}

	ip := protocol.NewIPv4(serverIP, dstIP, 0x11, datagram)
	packet, err := ip.MarshalBinary()
	if err != nil {
		return nil, err
	}

	eth := protocol.Ethernet{
		SrcMAC:  serverMAC,
		DstMAC:  dstMAC,
		Type:    0x0800,
		Payload: packet,
	}

	return eth.MarshalBinary()
}

func sendReply(ingress *network.Port, packet []byte) error {
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
