/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package device

import (
	"errors"
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"git.sds.co.kr/cherry.git/cherryd/openflow/of10"
	"golang.org/x/net/context"
	"net"
	"time"
)

type OF10Transceiver struct {
	baseTransceiver
}

func NewOF10Transceiver(stream *openflow.Stream, log Logger) *OF10Transceiver {
	return &OF10Transceiver{
		baseTransceiver: baseTransceiver{
			stream:  stream,
			log:     log,
			version: openflow.Ver10,
		},
	}
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

func (r *OF10Transceiver) addFlowMod(conf FlowModConfig) error {
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

func (r *OF10Transceiver) packetOut(inport openflow.InPort, action openflow.Action, data []byte) error {
	msg := of10.NewPacketOut(r.getTransactionID(), inport, action, data)
	return openflow.WriteMessage(r.stream, msg)
}

func (r *OF10Transceiver) newMatch() openflow.Match {
	return of10.NewMatch()
}

func (r *OF10Transceiver) newAction() openflow.Action {
	return new(of10.Action)
}

func (r *OF10Transceiver) handleFeaturesReply(msg *of10.FeaturesReply) error {
	r.device = findDevice(msg.DPID)
	r.device.NumBuffers = uint(msg.NumBuffers)
	r.device.NumTables = uint(msg.NumTables)
	r.device.addTransceiver(0, r)

	if msg.Capabilities.OFPC_STP == true {
		// TODO: Disable STP on all ports
	}
	for _, v := range msg.Ports {
		r.device.setPort(uint(v.Number()), v)
		// XXX: debugging
		r.log.Printf("Port: %+v", v)
	}

	// XXX: debugging
	{
		r.log.Printf("FeaturesReply: %+v", msg)

		getconfig := of10.NewGetConfigRequest(r.getTransactionID())
		if err := openflow.WriteMessage(r.stream, getconfig); err != nil {
			return err
		}

		match := r.newMatch()
		match.SetInPort(10)
		action := r.newAction()
		action.SetOutput(5)
		mac, err := net.ParseMAC("3c:07:54:6f:70:b5")
		if err != nil {
			panic("invalid MAC address")
		}
		action.SetDstMAC(mac)
		conf := FlowModConfig{
			IdleTimeout: 20,
			Priority:    10,
			Match:       match,
			Action:      action,
		}
		if err := r.addFlowMod(conf); err != nil {
			return err
		}

		action = r.newAction()
		action.SetOutput(15)
		lldp := []byte{0x01, 0x80, 0xc2, 0x00, 0x00, 0x0e, 0x00, 0x01, 0xe8, 0xd8, 0x0f, 0x32, 0x88, 0xcc, 0x02, 0x07, 0x04, 0x00, 0x01, 0xe8, 0xd8, 0x0f, 0x25, 0x04, 0x06, 0x05, 0x73, 0x77, 0x70, 0x31, 0x33, 0x06, 0x02, 0x00, 0x78, 0x0a, 0x07, 0x63, 0x75, 0x6d, 0x75, 0x6c, 0x75, 0x73, 0x0c, 0x34, 0x43, 0x75, 0x6d, 0x75, 0x6c, 0x75, 0x73, 0x20, 0x4c, 0x69, 0x6e, 0x75, 0x78, 0x20, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x20, 0x32, 0x2e, 0x35, 0x2e, 0x31, 0x20, 0x72, 0x75, 0x6e, 0x6e, 0x69, 0x6e, 0x67, 0x20, 0x6f, 0x6e, 0x20, 0x64, 0x6e, 0x69, 0x20, 0x65, 0x74, 0x2d, 0x37, 0x34, 0x34, 0x38, 0x62, 0x66, 0x0e, 0x04, 0x00, 0x14, 0x00, 0x14, 0x08, 0x05, 0x73, 0x77, 0x70, 0x31, 0x33, 0x00, 0x00}
		if err := r.packetOut(openflow.NewInPort(), action, lldp); err != nil {
			return err
		}
	}

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
	// Update port status
	r.device.setPort(msg.Port.Number(), msg.Port)

	// XXX: debugging
	{
		r.log.Printf("PortStatus: %+v, Port: %+v", msg, *msg.Port)
	}

	return nil
}

func (r *OF10Transceiver) handleFlowRemoved(msg *of10.FlowRemoved) error {
	// XXX: debugging
	{
		r.log.Printf("FlowRemoved: %+v, match=%+v", msg, msg.Match)
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
		Pool.remove(r.device.DPID)
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
