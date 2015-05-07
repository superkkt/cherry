/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package application

import (
	"fmt"
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

func handleARP(eth *protocol.Ethernet, ingress device.Point) (drop bool, err error) {
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

func makeICMPEchoReply(src, dst net.IP, req *protocol.ICMPEcho) ([]byte, error) {
	reply, err := protocol.NewICMPEchoReply(req.ID, req.Sequence, req.Payload).MarshalBinary()
	if err != nil {
		return nil, err
	}

	ip, err := protocol.NewIPv4(src, dst, 1, reply).MarshalBinary()
	if err != nil {
		return nil, err
	}

	return ip, nil
}

func handleIPv4(eth *protocol.Ethernet, ingress device.Point) (drop bool, err error) {
	fmt.Printf("Receiving IPv4 packet..\n")
	ip := new(protocol.IPv4)
	if err := ip.UnmarshalBinary(eth.Payload); err != nil {
		return false, err
	}
	fmt.Printf("IPv4: %+v\n", ip)

	// ICMP to 10.0.0.254?
	if ip.Protocol != 1 || !ip.DstIP.Equal(net.IPv4(10, 0, 0, 254)) {
		return false, nil
	}

	icmp := new(protocol.ICMPEcho)
	if err := icmp.UnmarshalBinary(ip.Payload); err != nil {
		return false, err
	}
	reply, err := makeICMPEchoReply(ip.DstIP, ip.SrcIP, icmp)
	if err != nil {
		return false, err
	}

	v, err := protocol.Ethernet{
		SrcMAC:  eth.DstMAC,
		DstMAC:  eth.SrcMAC,
		Type:    0x0800,
		Payload: reply,
	}.MarshalBinary()
	if err != nil {
		return false, err
	}

	action := ingress.Node.NewAction()
	action.SetOutput(uint(ingress.Port))

	return true, ingress.Node.PacketOut(openflow.NewInPort(), action, v)
}

func (r virtualRouter) run(eth *protocol.Ethernet, ingress device.Point) (drop bool, err error) {
	switch eth.Type {
	// ARP
	case 0x0806:
		return handleARP(eth, ingress)
	// IPv4
	case 0x0800:
		return handleIPv4(eth, ingress)
	default:
		return false, nil
	}

}
