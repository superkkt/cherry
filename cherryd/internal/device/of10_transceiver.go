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
	"time"
)

type OF10Transceiver struct {
	BaseTransceiver
}

func NewOF10Transceiver(stream *openflow.Stream, log Logger) *OF10Transceiver {
	return &OF10Transceiver{
		BaseTransceiver: BaseTransceiver{
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
	// XXX: debugging
	{
		r.log.Printf("Error: %+v", msg)
	}

	return nil
}

func (r *OF10Transceiver) handleMessage(msg openflow.Message) error {
	header := msg.Header()
	if header.Version != r.version {
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
	default:
		r.log.Printf("Unsupported message type: version=%v, type=%v", header.Version, header.Type)
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
	receivedMsg := make(chan openflow.Message)
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
