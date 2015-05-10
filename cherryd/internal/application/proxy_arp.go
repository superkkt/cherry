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
	Pool.add(newProxyARP())
}

type ProxyARP struct {
	baseProcessor
	routers map[string]net.IP
}

func newProxyARP() *ProxyARP {
	v := &ProxyARP{
		routers: make(map[string]net.IP),
	}

	// FIXME: Read router IP addresses from DB
	r1 := net.IPv4(223, 130, 122, 1)
	r2 := net.IPv4(223, 130, 123, 1)
	r3 := net.IPv4(223, 130, 124, 1)
	r4 := net.IPv4(223, 130, 125, 1)
	r5 := net.IPv4(10, 0, 0, 254)
	v.routers[r1.String()] = r1
	v.routers[r2.String()] = r2
	v.routers[r3.String()] = r3
	v.routers[r4.String()] = r4
	v.routers[r5.String()] = r5

	return v
}

func (r ProxyARP) name() string {
	return "ProxyARP"
}

func (r *ProxyARP) run(eth *protocol.Ethernet, ingress controller.Point) (drop bool, err error) {
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
		// XXX: debugging
		fmt.Printf("ARP Operation: %v\n", arp.Operation)
		return false, nil
	}
	_, ok := r.routers[arp.TPA.String()]
	if !ok {
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

func makeARPReply(request *protocol.ARP) ([]byte, error) {
	v := protocol.NewARPReply(virtualMAC, request.SHA, request.TPA, request.SPA)
	reply, err := v.MarshalBinary()
	if err != nil {
		return nil, err
	}
	eth := protocol.Ethernet{
		SrcMAC:  virtualMAC,
		DstMAC:  request.SHA,
		Type:    0x0806,
		Payload: reply,
	}

	return eth.MarshalBinary()
}
