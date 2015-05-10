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
	Pool.add(new(virtualRouter))
}

type virtualRouter struct {
	baseProcessor
}

func (r virtualRouter) name() string {
	return "VirtualRouter"
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

func handleICMP(eth *protocol.Ethernet, ingress controller.Point, ip *protocol.IPv4) (drop bool, err error) {
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

func (r virtualRouter) processPacket(eth *protocol.Ethernet, ingress controller.Point) (drop bool, err error) {
	// XXX: debugging
	fmt.Printf("VirtualRouter is running..\n")

	// IPv4 packet to the virtual router?
	if eth.Type != 0x0800 || eth.DstMAC.String() != virtualMAC.String() {
		return false, nil
	}

	ip := new(protocol.IPv4)
	if err := ip.UnmarshalBinary(eth.Payload); err != nil {
		return false, err
	}
	// XXX: debugging
	fmt.Printf("IPv4: %+v\n", ip)

	// ICMP?
	if ip.Protocol == 1 {
		return handleICMP(eth, ingress, ip)
	}

	// TODO: Do virtual routing

	return false, nil
}
