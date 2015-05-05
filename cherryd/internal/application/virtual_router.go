/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package application

import (
	"git.sds.co.kr/cherry.git/cherryd/internal/device"
	"git.sds.co.kr/cherry.git/cherryd/net/protocol"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"net"
)

var virtualRouterMAC net.HardwareAddr

func init() {
	Pool.add(new(virtualRouter))
	// Locally Administered Address
	v, err := net.ParseMAC("02:DB:CA:FE:00:01")
	if err != nil {
		panic("invalid MAC address")
	}
	virtualRouterMAC = v
}

type virtualRouter struct{}

func (r virtualRouter) name() string {
	return "Virtual Router"
}

func (r virtualRouter) priority() uint {
	return 10
}

func makeARPReply(request *protocol.ARP) ([]byte, error) {
	v := protocol.NewARPReply(virtualRouterMAC, request.SHA, request.TPA, request.SPA)
	reply, err := v.MarshalBinary()
	if err != nil {
		return nil, err
	}
	eth := protocol.Ethernet{
		SrcMAC:  virtualRouterMAC,
		DstMAC:  request.SHA,
		Type:    0x0806,
		Payload: reply,
	}

	return eth.MarshalBinary()
}

func (r virtualRouter) run(eth *protocol.Ethernet, ingress device.Point) (drop bool, err error) {
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
		return false, nil
	}
	if !arp.TPA.Equal(net.IPv4(10, 0, 0, 254)) {
		return false, nil
	}

	reply, err := makeARPReply(arp)
	if err != nil {
		return false, err
	}
	action := ingress.Node.NewAction()
	action.SetOutput(uint(ingress.Port))

	return true, ingress.Node.PacketOut(openflow.NewInPort(), action, reply)
}
