/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package l2switch

import (
	"bytes"
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"git.sds.co.kr/cherry.git/cherryd/internal/network"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"git.sds.co.kr/cherry.git/cherryd/protocol"
	"github.com/dlintw/goconf"
	"net"
)

type L2Switch struct {
	conf *goconf.ConfigFile
	log  log.Logger
}

func New(conf *goconf.ConfigFile, log log.Logger) *L2Switch {
	return &L2Switch{
		conf: conf,
		log:  log,
	}
}

func (r *L2Switch) Name() string {
	return "L2Switch"
}

func flood(f openflow.Factory, ingress *network.Port, packet []byte) error {
	inPort := openflow.NewInPort()
	inPort.SetPort(uint32(ingress.Number()))

	action, err := f.NewAction()
	if err != nil {
		return err
	}
	action.SetOutPort(openflow.OutFlood)

	out, err := f.NewPacketOut()
	if err != nil {
		return err
	}
	out.SetInPort(inPort)
	out.SetAction(action)
	out.SetData(packet)

	return ingress.Device().SendMessage(out)
}

func isBroadcast(eth *protocol.Ethernet) bool {
	return bytes.Compare(eth.DstMAC, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}) == 0
}

type flowParam struct {
	device    *network.Device
	etherType uint16
	inPort    uint
	outPort   uint
	srcMAC    net.HardwareAddr
	dstMAC    net.HardwareAddr
}

func installFlow(p flowParam) error {
	// TODO: Implement this function
	return nil
}

func setFlowRule(p flowParam) error {
	// Forward
	if err := installFlow(p); err != nil {
		return err
	}
	// Backward
	return installFlow(p)
}

func (r *L2Switch) ProcessPacket(factory openflow.Factory, finder network.Finder, eth *protocol.Ethernet, ingress *network.Port) (drop bool, err error) {
	r.log.Debug("L2Switch is executed..")

	packet, err := eth.MarshalBinary()
	if err != nil {
		return true, err
	}

	node := finder.Node(eth.DstMAC)
	// Unknown node or broadcast request?
	if node == nil || isBroadcast(eth) {
		r.log.Debug(fmt.Sprintf("Broadcasting.. dstMAC=%v", eth.DstMAC))
		return true, flood(factory, ingress, packet)
	}

	// On same switch device?
	if ingress.Device().ID() == node.Port().Device().ID() {
		param := flowParam{
			device:    ingress.Device(),
			etherType: eth.Type,
			inPort:    ingress.Number(),
			outPort:   node.Port().Number(),
			srcMAC:    eth.SrcMAC,
			dstMAC:    eth.DstMAC,
		}
		if err := setFlowRule(param); err != nil {
			return true, err
		}
	}

	path := finder.Path(ingress.Device().ID(), node.Port().Device().ID())
	if path == nil {
		r.log.Debug(fmt.Sprintf("Not found a path to %v", eth.DstMAC))
		return true, nil
	}
	inPort := ingress.Number()
	for _, p := range path {
		param := flowParam{
			device:    p[0].Device(),
			etherType: eth.Type,
			inPort:    inPort,
			outPort:   p[0].Number(),
			srcMAC:    eth.SrcMAC,
			dstMAC:    eth.DstMAC,
		}
		if err := setFlowRule(param); err != nil {
			return true, err
		}
		inPort = p[1].Number()
	}

	// TODO: Set PACKET_OUT for the final device

	return false, nil
}

func (r *L2Switch) ProcessEvent() error {
	// TODO:  Remove all flows that have MAC addresses, which are vanished, in its source or destination
	return nil
}
