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

func (r *l2Switch) run(eth *protocol.Ethernet, ingress device.Point) (drop bool, err error) {
	flood := func(port uint32, data []byte) error {
		// XXX: debugging
		fmt.Print("Flooding..")

		// Flooding
		v := openflow.NewInPort()
		v.SetPort(uint(port))
		return ingress.Node.Flood(v, data)
	}

	// FIXME: Is it better to get the raw packet as an input parameter?
	packet, err := eth.MarshalBinary()
	if err != nil {
		return false, err
	}

	if isARPRequest(eth) {
		// XXX: debugging
		fmt.Print("ARP request is received..\n")
		return true, flood(ingress.Port, packet)
	}

	// TODO: Add test cases for hosts DB
	dstPoint, ok := device.Hosts.Find(eth.DstMAC)
	if !ok {
		// XXX: debugging
		fmt.Printf("Failed to find the destination MAC: %v\n", eth.DstMAC)
		return true, flood(ingress.Port, packet)
	}
	path := device.Switches.FindPath(ingress.Node, dstPoint.Node)
	// Empty path means the destination is not connected with this device that sent PACKET_IN.
	if len(path) == 0 {
		// XXX: debugging
		fmt.Printf("We don't know the path to the destintion: %v\n", eth.DstMAC)
		// FIXME: Flood? or Drop?
		return true, flood(ingress.Port, packet)
	}
	// Install flows bidirectionally in all switches on the path
	inPort := ingress.Port
	for _, v := range path {
		src, dst := getPoint(v)
		installFlowRule(inPort, eth.SrcMAC, eth.DstMAC, src)
		installFlowRule(src.Port, eth.DstMAC, eth.SrcMAC, &device.Point{src.Node, inPort})
		inPort = dst.Port
	}
	// TODO: Remove flows when the port, which is used in the flow, is removed

	// XXX: debugging
	fmt.Printf("Sending PACKET_OUT to %v..\n", eth.DstMAC)

	// PacketOut on the final destination switch
	action := dstPoint.Node.NewAction()
	action.SetOutput(uint(dstPoint.Port))
	return true, dstPoint.Node.PacketOut(openflow.NewInPort(), action, packet)
}

func getPoint(path graph.Path) (src, dst *device.Point) {
	node := path.V.(*device.Device)
	edge := path.E.(*device.Edge)
	if edge.P1.Node.ID() == node.ID() {
		return edge.P1, edge.P2
	}
	return edge.P2, edge.P1
}

func installFlowRule(inPort uint32, src, dst net.HardwareAddr, p *device.Point) error {
	match := p.Node.NewMatch()
	match.SetInPort(inPort)
	match.SetSrcMAC(src)
	match.SetDstMAC(dst)
	return _installFlowRule(match, p)
}

func _installFlowRule(match openflow.Match, p *device.Point) error {
	action := p.Node.NewAction()
	action.SetOutput(uint(p.Port))
	c := device.FlowModConfig{
		IdleTimeout: 30,
		Priority:    10,
		Match:       match,
		Action:      action,
	}

	return p.Node.InstallFlowRule(c)
}
