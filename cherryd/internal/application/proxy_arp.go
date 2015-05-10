/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package application

import (
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/internal/controller"
	"git.sds.co.kr/cherry.git/cherryd/net/protocol"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"net"
)

func init() {
	Pool.add(new(proxyARP))
}

type proxyARP struct {
	baseProcessor
}

func (r proxyARP) name() string {
	return "ProxyARP"
}

func (r *proxyARP) processPacket(eth *protocol.Ethernet, ingress controller.Point) (drop bool, err error) {
	// XXX: debugging
	fmt.Printf("ProxyARP is running..\n")

	// ARP?
	if eth.Type != 0x0806 {
		return false, nil
	}

	arp := new(protocol.ARP)
	if err := arp.UnmarshalBinary(eth.Payload); err != nil {
		return false, err
	}
	// ARP request?
	if arp.Operation != 1 {
		// Drop all ARP packets if they are not a reqeust.
		return true, nil
	}
	// Pass ARP announcements packets if it has valid source IP & MAC addresses
	if isARPAnnouncement(arp) {
		return r.handleARPAnnouncement(arp)
	}

	mac, ok := hostDB.MAC(arp.TPA)
	if !ok {
		// Unknown hosts. Drop the packet.
		return true, nil
	}

	reply, err := makeARPReply(arp, mac)
	if err != nil {
		return false, err
	}
	action := ingress.Node.NewAction()
	action.SetOutput(uint(ingress.Port))

	return true, ingress.Node.PacketOut(openflow.NewInPort(), action, reply)
}

func isARPAnnouncement(request *protocol.ARP) bool {
	// Valid ARP announcement?
	sameAddr := request.SPA.Equal(request.TPA)
	zeroTarget := request.THA.String() == "00:00:00:00:00:00"
	if !sameAddr || !zeroTarget {
		return false
	}

	return true
}

func (r *proxyARP) handleARPAnnouncement(request *protocol.ARP) (drop bool, err error) {
	// Trusted MAC address?
	mac, ok := hostDB.MAC(request.SPA)
	if !ok || mac.String() != request.SHA.String() {
		// Drop suspicious announcements
		return true, nil
	}

	return false, nil
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
