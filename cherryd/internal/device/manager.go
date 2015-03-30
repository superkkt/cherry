/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

// Device means an OpenFlow switch, but the word "switch" is usually reserved
// for conditional statements in most programming languages. So, we use "device"
// instead of "switch" in this project.
package device

import (
	"fmt"
	"git.sds.co.kr/bosomi.git/socket"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"golang.org/x/net/context"
	"log"
	"net"
	"time"
)

const (
	socketTimeout = 5 * time.Second
)

type Manager struct {
	log          *log.Logger
	openflow     *openflow.Transceiver
	DPID         uint64
	NumBuffers   uint32
	NumTables    uint8
	Capabilities *openflow.Capability
	Actions      *openflow.Action
	Ports        []openflow.Port
}

func NewManager(log *log.Logger) *Manager {
	return &Manager{
		log: log,
	}
}

// TODO: Add functions than will be called by a plugin application,
// e.g., GetDeviceDescription(), GetDeviceFeatures(), etc., which calls
// counterpart functions in the openflow package

func (r *Manager) handleHelloMessage(msg *openflow.HelloMessage) error {
	// We only support OF 1.0
	if msg.Version < 0x01 {
		fmt.Errorf("unsupported OpenFlow protocol version: 0x%X", msg.Version)
	}

	if err := r.openflow.SendDescStatsRequestMessage(); err != nil {
		return err
	}
	if err := r.openflow.SendFeaturesRequestMessage(); err != nil {
		return err
	}

	return nil
}

func (r *Manager) handleErrorMessage(msg *openflow.ErrorMessage) error {
	r.log.Printf("error from a device: dpid=%v, type=%v, code=%v, data=%v",
		r.DPID, msg.Type, msg.Code, msg.Data)
	return nil
}

func (r *Manager) handleFeaturesReplyMessage(msg *openflow.FeaturesReplyMessage) error {
	// XXX: debugging
	r.log.Printf("DPID: %v", msg.DPID)
	r.log.Printf("# of buffers: %v", msg.NumBuffers)
	r.log.Printf("# of tables: %v", msg.NumTables)
	r.log.Printf("Capabilities: %+v", msg.GetCapability())
	r.log.Printf("Actions: %+v", msg.GetSupportedAction())
	for _, v := range msg.Ports {
		r.log.Printf("No: %v, MAC: %v, Name: %v, Port Down?: %v, Link Down?: %v, Current: %+v, Advertised: %+v, Supported: %+v", v.Number, v.MAC, v.Name, v.IsPortDown(), v.IsLinkDown(), v.GetCurrentFeatures(), v.GetAdvertisedFeatures(), v.GetSupportedFeatures())
	}

	r.DPID = msg.DPID
	r.NumBuffers = msg.NumBuffers
	r.NumTables = msg.NumTables
	r.Capabilities = msg.GetCapability()
	r.Actions = msg.GetSupportedAction()
	r.Ports = msg.Ports

	// Add this device to the device pool
	add(r.DPID, r)

	// XXX: test
	match := openflow.NewFlowMatch()
	match.SetInPort(39)
	match.SetSrcIP(net.ParseIP("223.130.120.0"), 8)
	match.SetDstIP(net.ParseIP("223.130.122.0"), 8)
	a1 := &openflow.FlowActionOutput{Port: 40}
	a2 := &openflow.FlowActionOutput{Port: 41}
	a3 := &openflow.FlowActionOutput{Port: 42}
	rule := FlowRule{
		Match:       match,
		IdleTimeout: 30,
		Actions:     []openflow.FlowAction{a1, a2, a3},
	}
	//if err := r.RemoveFlowRule(match); err != nil {
	if err := r.InstallFlowRule(rule); err != nil {
		r.log.Printf("failed to install a flow rule: %v", err)
	}
	if err := r.openflow.SendFlowStatsRequestMessage(openflow.NewFlowMatch()); err != nil {
		r.log.Printf("failed to send a flow_stats_request: %v", err)
	}

	return nil
}

func (r *Manager) handleEchoRequestMessage(msg *openflow.EchoRequestMessage) error {
	// XXX: debugging
	r.log.Printf("%+v", msg)
	return nil
}

func (r *Manager) handleEchoReplyMessage(msg *openflow.EchoReplyMessage) error {
	// XXX: debugging
	r.log.Printf("%+v", msg)
	return nil
}

// TODO: Test this function by plug and unplug a port
func (r *Manager) handlePortStatusMessage(msg *openflow.PortStatusMessage) error {
	// Update port status
	for i, v := range r.Ports {
		if v.Number != msg.Target.Number {
			continue
		}
		r.Ports[i] = msg.Target
		// XXX: debugging
		r.log.Printf("Device Port Status: %+v", r.Ports[i])
	}

	// XXX: debugging
	r.log.Printf("%+v", msg)
	return nil
}

func (r *Manager) handlePacketInMessage(msg *openflow.PacketInMessage) error {
	// XXX: debugging
	r.log.Printf("%+v", msg)

	// XXX: test
	inPort := openflow.PortNumber(msg.InPort)
	flood := &openflow.FlowActionOutput{Port: openflow.OFPP_ALL}
	if err := r.SendPacketOut(inPort, []openflow.FlowAction{flood}, msg.Data); err != nil {
		r.log.Printf("failed to send a packet-out message: %v", err)
	}

	return nil
}

func (r *Manager) handleFlowRemovedMessage(msg *openflow.FlowRemovedMessage) error {
	// XXX: debugging
	r.log.Printf("%+v", msg)
	return nil
}

func (r *Manager) handleDescStatsReplyMessage(msg *openflow.DescStatsReplyMessage) error {
	// XXX: debugging
	r.log.Printf("%+v", msg)
	return nil
}

func (r *Manager) handleFlowStatsReplyMessage(msg *openflow.FlowStatsReplyMessage) error {
	// XXX: debugging
	r.log.Printf("%+v", msg)
	for _, v := range msg.Flows {
		r.log.Printf("%+v", v)
		r.log.Printf("%+v", v.Match)
		r.log.Printf("%+v", v.Match.GetFlowWildcards())
		srcIP, bits := v.Match.GetSrcIP()
		r.log.Printf("src_ip: %v, bits: %v", srcIP, bits)
		dstIP, bits := v.Match.GetDstIP()
		r.log.Printf("dst_ip: %v, bits: %v", dstIP, bits)
		for _, a := range v.Actions {
			r.log.Printf("%+v", a)
		}
	}
	return nil
}

func (r *Manager) Run(ctx context.Context, conn net.Conn) {
	socket := socket.NewConn(conn, 65535) // 65535 bytes are max size of a OpenFlow packet
	config := openflow.Config{
		Log:          r.log,
		Socket:       socket,
		ReadTimeout:  socketTimeout,
		WriteTimeout: socketTimeout,
		Handlers: openflow.MessageHandler{
			HelloMessage:          r.handleHelloMessage,
			ErrorMessage:          r.handleErrorMessage,
			FeaturesReplyMessage:  r.handleFeaturesReplyMessage,
			EchoRequestMessage:    r.handleEchoRequestMessage,
			EchoReplyMessage:      r.handleEchoReplyMessage,
			PortStatusMessage:     r.handlePortStatusMessage,
			PacketInMessage:       r.handlePacketInMessage,
			FlowRemovedMessage:    r.handleFlowRemovedMessage,
			DescStatsReplyMessage: r.handleDescStatsReplyMessage,
			FlowStatsReplyMessage: r.handleFlowStatsReplyMessage,
		},
	}

	of, err := openflow.NewTransceiver(config)
	if err != nil {
		r.log.Print(err)
	}
	r.openflow = of
	r.openflow.Run(ctx)

	// Remove this device from the device pool
	remove(r.DPID)
}

type FlowRule struct {
	Match       *openflow.FlowMatch
	Actions     []openflow.FlowAction
	IdleTimeout uint16
	HardTimeout uint16
}

// FIXME: Should we need to install a barrier after installing a flow rule?
func (r *Manager) InstallFlowRule(flow FlowRule) error {
	mod := &openflow.FlowModifyMessage{
		Match:       flow.Match,
		Command:     openflow.OFPFC_ADD,
		IdleTimeout: flow.IdleTimeout,
		HardTimeout: flow.HardTimeout,
		Flags: openflow.FlowModifyFlag{
			SendFlowRemoved: true,
			CheckOverlap:    true,
		},
		Actions: flow.Actions,
	}
	return r.openflow.SendFlowModifyMessage(mod)
}

func (r *Manager) RemoveFlowRule(match *openflow.FlowMatch) error {
	mod := &openflow.FlowModifyMessage{
		Match:   match,
		Command: openflow.OFPFC_DELETE,
	}
	return r.openflow.SendFlowModifyMessage(mod)
}

func (r *Manager) SendPacketOut(inPort openflow.PortNumber, actions []openflow.FlowAction, packet []byte) error {
	return r.openflow.SendPacketOutMessage(inPort, actions, packet)
}
