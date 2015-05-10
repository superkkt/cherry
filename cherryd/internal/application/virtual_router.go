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

var vMAC net.HardwareAddr
var vIP net.IP

func init() {
	Pool.add(new(virtualRouter))
	// Locally Administered Address
	mac, err := net.ParseMAC("02:DB:CA:FE:00:01")
	if err != nil {
		panic("invalid MAC address")
	}
	vMAC = mac
	vIP = net.IPv4(223, 130, 122, 1)
}

type virtualRouter struct {
	prior uint
	state bool
}

func (r virtualRouter) name() string {
	return "VirtualRouter"
}

func (r virtualRouter) priority() uint {
	return r.prior
}

func (r *virtualRouter) setPriority(p uint) {
	r.prior = p
}

func (r *virtualRouter) enable() {
	r.state = true
}

func (r *virtualRouter) disable() {
	r.state = false
}

func (r virtualRouter) enabled() bool {
	return r.state
}

func makeARPReply(request *protocol.ARP) ([]byte, error) {
	v := protocol.NewARPReply(vMAC, request.SHA, request.TPA, request.SPA)
	reply, err := v.MarshalBinary()
	if err != nil {
		return nil, err
	}
	eth := protocol.Ethernet{
		SrcMAC:  vMAC,
		DstMAC:  request.SHA,
		Type:    0x0806,
		Payload: reply,
	}

	return eth.MarshalBinary()
}

func handleARP(eth *protocol.Ethernet, ingress controller.Point) (drop bool, err error) {
	fmt.Printf("Receiving ARP packet..\n")

	arp := new(protocol.ARP)
	if err := arp.UnmarshalBinary(eth.Payload); err != nil {
		return false, err
	}
	// ARP request?
	if arp.Operation != 1 {
		fmt.Printf("ARP Operation: %v\n", arp.Operation)
		return false, nil
	}
	fmt.Printf("vIP: %v, ARP TPA: %v\n", vIP, arp.TPA)
	if !arp.TPA.Equal(vIP) {
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

func handleIPv4(eth *protocol.Ethernet, ingress controller.Point) (drop bool, err error) {
	fmt.Printf("Receiving IPv4 packet..\n")

	ip := new(protocol.IPv4)
	if err := ip.UnmarshalBinary(eth.Payload); err != nil {
		return false, err
	}
	fmt.Printf("IPv4: %+v\n", ip)

	// ICMP?
	if ip.Protocol != 1 || !ip.DstIP.Equal(vIP) {
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

func (r virtualRouter) run(eth *protocol.Ethernet, ingress controller.Point) (drop bool, err error) {
	fmt.Printf("VirtualRouter is running..\n")

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
