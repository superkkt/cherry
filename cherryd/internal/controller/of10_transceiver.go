/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package controller

import (
	"errors"
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"git.sds.co.kr/cherry.git/cherryd/openflow/of10"
	"golang.org/x/net/context"
	"time"
)

type OF10Transceiver struct {
	baseTransceiver
}

func NewOF10Transceiver(stream *openflow.Stream, log Logger, p Processor) *OF10Transceiver {
	v := &OF10Transceiver{
		baseTransceiver: baseTransceiver{
			stream:    stream,
			log:       log,
			version:   openflow.Ver10,
			processor: p,
		},
	}
	v.lldpExplored.Store(false)
	return v
}

func (r *OF10Transceiver) sendHello() error {
	hello := openflow.NewHello(r.version, r.getTransactionID())
	return openflow.WriteMessage(r.stream, hello)
}

func (r *OF10Transceiver) sendFeaturesRequest() error {
	feature := of10.NewFeaturesRequest(r.getTransactionID())
	return openflow.WriteMessage(r.stream, feature)
}

func (r *OF10Transceiver) sendBarrierRequest() error {
	barrier := of10.NewBarrierRequest(r.getTransactionID())
	return openflow.WriteMessage(r.stream, barrier)
}

func (r *OF10Transceiver) sendSetConfig(flags, missSendLen uint16) error {
	msg := of10.NewSetConfig(r.getTransactionID(), flags, missSendLen)
	return openflow.WriteMessage(r.stream, msg)
}

func (r *OF10Transceiver) sendDescriptionRequest() error {
	msg := of10.NewDescriptionRequest(r.getTransactionID())
	return openflow.WriteMessage(r.stream, msg)
}

func (r *OF10Transceiver) addFlow(conf FlowModConfig) error {
	c := &of10.FlowModConfig{
		// TODO: set Cookie
		IdleTimeout: conf.IdleTimeout,
		HardTimeout: conf.HardTimeout,
		Priority:    conf.Priority,
		Match:       conf.Match,
		Action:      conf.Action,
	}
	msg := of10.NewFlowModAdd(r.getTransactionID(), c)
	return openflow.WriteMessage(r.stream, msg)
}

func (r *OF10Transceiver) removeFlow(match openflow.Match) error {
	c := &of10.FlowModConfig{
		Match: match,
	}

	return openflow.WriteMessage(r.stream, of10.NewFlowModDelete(r.getTransactionID(), c))
}

func (r *OF10Transceiver) packetOut(inport openflow.InPort, action openflow.Action, data []byte) error {
	msg := of10.NewPacketOut(r.getTransactionID(), inport, action, data)
	return openflow.WriteMessage(r.stream, msg)
}

func (r *OF10Transceiver) flood(inPort openflow.InPort, data []byte) error {
	if r.device == nil {
		panic("flood on nil device")
	}

	action := of10.NewAction()
	action.SetOutput(of10.OFPP_FLOOD)
	msg := of10.NewPacketOut(r.getTransactionID(), inPort, action, data)
	return openflow.WriteMessage(r.stream, msg)
}

func (r *OF10Transceiver) newMatch() openflow.Match {
	return of10.NewMatch()
}

func (r *OF10Transceiver) newAction() openflow.Action {
	return of10.NewAction()
}

func (r *OF10Transceiver) handleFeaturesReply(msg *of10.FeaturesReply) error {
	v := Switches.Get(msg.DPID)
	if v == nil {
		v = newDevice(msg.DPID)
	}
	Switches.add(msg.DPID, v)

	r.device = v
	r.device.NumBuffers = uint(msg.NumBuffers)
	r.device.NumTables = uint(msg.NumTables)
	r.device.addTransceiver(0, r)
	r.connected = true

	for _, v := range msg.Ports {
		// Reserved port?
		if v.Number() > of10.OFPP_MAX {
			continue
		}
		// Add new port information
		r.device.setPort(uint(v.Number()), v)
		if err := r.sendLLDP(v); err != nil {
			return err
		}
		// XXX: debugging
		r.log.Printf("Port: %+v", v)
	}
	r.startLLDPTimer()

	return nil
}

func (r *OF10Transceiver) handleGetConfigReply(msg *of10.GetConfigReply) error {
	// XXX: debugging
	{
		r.log.Printf("GetConfigReply: %+v", msg)
	}

	return nil
}

func (r *OF10Transceiver) handleDescriptionReply(msg *of10.DescriptionReply) error {
	r.device.Manufacturer = msg.Manufacturer
	r.device.Hardware = msg.Hardware
	r.device.Software = msg.Software
	r.device.Serial = msg.Serial
	r.device.Description = msg.Description

	// XXX: debugging
	{
		r.log.Printf("DescriptionReply: %+v", msg)
	}

	return nil
}

func (r *OF10Transceiver) handleError(msg *openflow.Error) error {
	r.log.Printf("Error: version=%v, xid=%v, type=%v, code=%v", msg.Version(), msg.TransactionID(), msg.Class, msg.Code)
	return nil
}

func (r *OF10Transceiver) handlePortStatus(msg *of10.PortStatus) error {
	if r.device == nil {
		r.log.Print("PortStatus is received, but we don't have a switch device yet!")
		return nil
	}

	if err := r.sendPortStatusEvent(msg.Port); err != nil {
		return err
	}
	return r.updatePortStatus(msg.Port)
}

func (r *OF10Transceiver) handleFlowRemoved(msg *of10.FlowRemoved) error {
	// XXX: debugging
	{
		r.log.Printf("FlowRemoved: %+v, match=%+v", msg, msg.Match)
	}

	return nil
}

func (r *OF10Transceiver) handlePacketIn(msg *of10.PacketIn) error {
	if !r.connected {
		return nil
	}
	r.handleIncoming(uint32(msg.InPort), msg.Data)

	// XXX: debugging
	{
		r.log.Printf("PacketIn: %+v", msg)
	}

	return nil
}

func (r *OF10Transceiver) handleMessage(msg openflow.Incoming) error {
	if msg.Version() != r.version {
		return errors.New("unexpected openflow protocol version!")
	}

	switch v := msg.(type) {
	case *openflow.EchoRequest:
		return r.handleEchoRequest(v)
	case *openflow.EchoReply:
		return r.handleEchoReply(v)
	case *openflow.Error:
		return r.handleError(v)
	case *of10.FeaturesReply:
		return r.handleFeaturesReply(v)
	case *of10.GetConfigReply:
		return r.handleGetConfigReply(v)
	case *of10.DescriptionReply:
		return r.handleDescriptionReply(v)
	case *of10.PortStatus:
		return r.handlePortStatus(v)
	case *of10.FlowRemoved:
		return r.handleFlowRemoved(v)
	case *of10.PacketIn:
		return r.handlePacketIn(v)
	default:
		r.log.Printf("Unsupported message type: version=%v, type=%v", msg.Version(), msg.Type())
		return nil
	}
}

func (r *OF10Transceiver) cleanup() {
	if r.device == nil {
		return
	}

	if r.device.removeTransceiver(0) == 0 {
		r.baseTransceiver.cleanup()
	}
}

func (r *OF10Transceiver) init() error {
	if err := r.sendHello(); err != nil {
		return fmt.Errorf("failed to send hello message: %v", err)
	}
	if err := r.sendSetConfig(of10.OFPC_FRAG_NORMAL, 0xFFFF); err != nil {
		return fmt.Errorf("failed to send set_config message: %v", err)
	}
	if err := r.sendFeaturesRequest(); err != nil {
		return fmt.Errorf("failed to send features_request message: %v", err)
	}
	if err := r.sendDescriptionRequest(); err != nil {
		return fmt.Errorf("failed to send description_request message: %v", err)
	}
	if err := r.sendBarrierRequest(); err != nil {
		return fmt.Errorf("failed to send barrier_request: %v", err)
	}

	return nil
}

func (r *OF10Transceiver) Run(ctx context.Context) {
	defer r.cleanup()

	r.stream.SetReadTimeout(1 * time.Second)
	r.stream.SetWriteTimeout(5 * time.Second)
	if err := r.init(); err != nil {
		r.log.Printf("init: %v", err)
		return
	}
	go r.pinger(ctx, r.version)

	// Reader goroutine
	receivedMsg := make(chan openflow.Incoming)
	go func() {
		for {
			msg, err := openflow.ReadMessage(r.stream)
			if err != nil {
				switch {
				case openflow.IsTimeout(err):
					// Ignore timeout error
					continue
				case err == openflow.ErrUnsupportedMessage:
					r.log.Print(err)
					continue
				default:
					r.log.Print(err)
					close(receivedMsg)
					return
				}
			}
			receivedMsg <- msg
		}
	}()

	// Infinite loop
	for {
		select {
		case msg, ok := <-receivedMsg:
			if !ok {
				return
			}
			if err := r.handleMessage(msg); err != nil {
				r.log.Print(err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
