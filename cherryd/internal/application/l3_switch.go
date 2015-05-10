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
	Pool.add(new(L3Switch))
}

type L3Switch struct {
	baseProcessor
}

func (r L3Switch) name() string {
	return "L3Switch"
}

func (r *L3Switch) run(eth *protocol.Ethernet, ingress controller.Point) (drop bool, err error) {
	packet, err := eth.MarshalBinary()
	if err != nil {
		return false, err
	}

	if isBroadcast(eth) {
		// XXX: debugging
		fmt.Print("Broadcasting..\n")
		return true, flood(ingress.Node, ingress.Port, packet)
	}

	destination, ok := controller.Hosts.Find(eth.DstMAC)
	if !ok {
		// XXX: debugging
		fmt.Printf("Failed to find the destination MAC: %v\n", eth.DstMAC)
		return true, flood(ingress.Node, ingress.Port, packet)
	}

	if err := r.installFlowRule(eth, ingress, destination); err != nil {
		return false, err
	}

	// XXX: debugging
	fmt.Printf("Sending PACKET_OUT to %v..\n", eth.DstMAC)

	// PacketOut on the final destination switch
	action := destination.Node.NewAction()
	action.SetOutput(uint(destination.Port))
	return true, destination.Node.PacketOut(openflow.NewInPort(), action, packet)
}

func (r L3Switch) installFlowRule(eth *protocol.Ethernet, ingress, destination controller.Point) error {
	// XXX: HP 2920 only does not support Dst. MAC as a packet matching column,
	// so we implement this L2 MAC learning switch based on L3 IP addresses instead of L2 MAC addresses.
	if eth.Type != 0x0800 {
		return nil
	}
	ip := new(protocol.IPv4)
	if err := ip.UnmarshalBinary(eth.Payload); err != nil {
		return err
	}
	srcIP := &net.IPNet{
		IP:   ip.SrcIP,
		Mask: net.IPv4Mask(255, 255, 255, 255),
	}
	dstIP := &net.IPNet{
		IP:   ip.DstIP,
		Mask: net.IPv4Mask(255, 255, 255, 255),
	}

	// src and dst nodes are on same node?
	if ingress.Node.DPID == destination.Node.DPID {
		if err := r._installFlowRule(ingress.Port, eth.Type, srcIP, dstIP, &destination); err != nil {
			return err
		}
		return r._installFlowRule(destination.Port, eth.Type, dstIP, srcIP, &ingress)
	}

	path := controller.Switches.FindPath(ingress.Node, destination.Node)
	// Empty path means the destination is not connected with this device that sent PACKET_IN.
	if len(path) == 0 {
		// XXX: debugging
		fmt.Printf("We don't know the path to the destintion: %v\n", eth.DstMAC)
		// FIXME: Is this an error?
		return nil
	}
	// Install flows bidirectionally in all switches on the path
	inPort := ingress.Port
	for _, v := range path {
		src, dst := getPoint(v)
		if err := r._installFlowRule(inPort, eth.Type, srcIP, dstIP, src); err != nil {
			return err
		}
		if err := r._installFlowRule(src.Port, eth.Type, dstIP, srcIP, &controller.Point{src.Node, inPort}); err != nil {
			return err
		}
		inPort = dst.Port
	}

	return nil
}

func (r L3Switch) _installFlowRule(inPort uint32, etherType uint16, srcIP, dstIP *net.IPNet, destination *controller.Point) error {
	match := destination.Node.NewMatch()
	match.SetInPort(inPort)
	match.SetEtherType(etherType)
	match.SetSrcIP(srcIP)
	match.SetDstIP(dstIP)

	action := destination.Node.NewAction()
	action.SetOutput(uint(destination.Port))
	c := controller.FlowModConfig{
		IdleTimeout: 30,
		Priority:    10,
		Match:       match,
		Action:      action,
	}

	return destination.Node.InstallFlowRule(c)
}
