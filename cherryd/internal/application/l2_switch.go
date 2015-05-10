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
	"git.sds.co.kr/cherry.git/cherryd/internal/graph"
	"git.sds.co.kr/cherry.git/cherryd/net/protocol"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"net"
	"strings"
)

func init() {
	Pool.add(new(L2Switch))
}

type L2Switch struct {
	baseProcessor
}

func (r *L2Switch) name() string {
	return "L2Switch"
}

func isBroadcast(eth *protocol.Ethernet) bool {
	return strings.ToUpper(eth.DstMAC.String()) == "FF:FF:FF:FF:FF:FF"
}

func flood(node *controller.Device, port uint32, data []byte) error {
	// XXX: debugging
	fmt.Print("Flooding..\n")

	// Flooding
	v := openflow.NewInPort()
	v.SetPort(uint(port))
	return node.Flood(v, data)
}

func (r *L2Switch) run(eth *protocol.Ethernet, ingress controller.Point) (drop bool, err error) {
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

func getPoint(path graph.Path) (src, dst *controller.Point) {
	node := path.V.(*controller.Device)
	edge := path.E.(*controller.Edge)
	if edge.P1.Node.ID() == node.ID() {
		return edge.P1, edge.P2
	}
	return edge.P2, edge.P1
}

func (r L2Switch) installFlowRule(eth *protocol.Ethernet, ingress, destination controller.Point) error {
	// src and dst nodes are on same node?
	if ingress.Node.DPID == destination.Node.DPID {
		if err := r._installFlowRule(ingress.Port, eth.Type, eth.SrcMAC, eth.DstMAC, &destination); err != nil {
			return err
		}
		return r._installFlowRule(destination.Port, eth.Type, eth.DstMAC, eth.SrcMAC, &ingress)
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
		if err := r._installFlowRule(inPort, eth.Type, eth.SrcMAC, eth.DstMAC, src); err != nil {
			return err
		}
		if err := r._installFlowRule(src.Port, eth.Type, eth.DstMAC, eth.SrcMAC, &controller.Point{src.Node, inPort}); err != nil {
			return err
		}
		inPort = dst.Port
	}

	return nil
}

func (r L2Switch) _installFlowRule(inPort uint32, etherType uint16, srcMAC, dstMAC net.HardwareAddr, destination *controller.Point) error {
	match := destination.Node.NewMatch()
	match.SetInPort(inPort)
	match.SetEtherType(etherType)
	match.SetSrcMAC(srcMAC)
	match.SetDstMAC(dstMAC)

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
