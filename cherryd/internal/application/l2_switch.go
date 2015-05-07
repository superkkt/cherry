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
	"git.sds.co.kr/cherry.git/cherryd/internal/graph"
	"git.sds.co.kr/cherry.git/cherryd/net/protocol"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"net"
	"strings"
)

func init() {
	Pool.add(new(l2Switch))
}

type l2Switch struct{}

func (r *l2Switch) name() string {
	return "L2 MAC Learning Switch"
}

func (r *l2Switch) priority() uint {
	return 100
}

func isARPRequest(eth *protocol.Ethernet) bool {
	return eth.Type == 0x0806 && strings.ToUpper(eth.DstMAC.String()) == "FF:FF:FF:FF:FF:FF"
}

func flood(node *device.Device, port uint32, data []byte) error {
	// XXX: debugging
	fmt.Print("Flooding..\n")

	// Flooding
	v := openflow.NewInPort()
	v.SetPort(uint(port))
	return node.Flood(v, data)
}

// TODO: Remove flows when the port, which is used in the flow, is removed
func (r *l2Switch) run(eth *protocol.Ethernet, ingress device.Point) (drop bool, err error) {
	// FIXME: Is it better to get the raw packet as an input parameter?
	packet, err := eth.MarshalBinary()
	if err != nil {
		return false, err
	}

	if isARPRequest(eth) {
		// XXX: debugging
		fmt.Print("ARP request is received..\n")
		return true, flood(ingress.Node, ingress.Port, packet)
	}

	// TODO: Add test cases for hosts DB
	destination, ok := device.Hosts.Find(eth.DstMAC)
	if !ok {
		// XXX: debugging
		fmt.Printf("Failed to find the destination MAC: %v\n", eth.DstMAC)
		return true, flood(ingress.Node, ingress.Port, packet)
	}

	if err := installFlowRule(eth, ingress, destination); err != nil {
		return false, err
	}

	// XXX: debugging
	fmt.Printf("Sending PACKET_OUT to %v..\n", eth.DstMAC)

	// PacketOut on the final destination switch
	action := destination.Node.NewAction()
	action.SetOutput(uint(destination.Port))
	return true, destination.Node.PacketOut(openflow.NewInPort(), action, packet)
}

func getPoint(path graph.Path) (src, dst *device.Point) {
	node := path.V.(*device.Device)
	edge := path.E.(*device.Edge)
	if edge.P1.Node.ID() == node.ID() {
		return edge.P1, edge.P2
	}
	return edge.P2, edge.P1
}

func installFlowRule(eth *protocol.Ethernet, ingress, destination device.Point) error {
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
		if err := _installFlowRule(ingress.Port, eth.Type, srcIP, dstIP, &destination); err != nil {
			return err
		}
		return _installFlowRule(destination.Port, eth.Type, dstIP, srcIP, &ingress)
	}

	path := device.Switches.FindPath(ingress.Node, destination.Node)
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
		if err := _installFlowRule(inPort, eth.Type, srcIP, dstIP, src); err != nil {
			return err
		}
		if err := _installFlowRule(src.Port, eth.Type, dstIP, srcIP, &device.Point{src.Node, inPort}); err != nil {
			return err
		}
		inPort = dst.Port
	}

	return nil
}

func _installFlowRule(inPort uint32, etherType uint16, srcIP, dstIP *net.IPNet, destination *device.Point) error {
	match := destination.Node.NewMatch()
	match.SetInPort(inPort)
	match.SetEtherType(etherType)
	match.SetSrcIP(srcIP)
	match.SetDstIP(dstIP)

	action := destination.Node.NewAction()
	action.SetOutput(uint(destination.Port))
	c := device.FlowModConfig{
		IdleTimeout: 30,
		Priority:    10,
		Match:       match,
		Action:      action,
	}

	return destination.Node.InstallFlowRule(c)
}
