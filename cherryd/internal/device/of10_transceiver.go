/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package device

import (
	"errors"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"git.sds.co.kr/cherry.git/cherryd/openflow/of10"
	"golang.org/x/net/context"
	"time"
)

type OF10Transceiver struct {
	BaseTransceiver
	version uint8
}

func NewOF10Transceiver(stream *openflow.Stream, log Logger) *OF10Transceiver {
	return &OF10Transceiver{
		BaseTransceiver: BaseTransceiver{
			stream: stream,
			log:    log,
		},
		version: openflow.Ver10,
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

func (r *OF10Transceiver) handleFeaturesReply(msg openflow.Message) error {
	reply, ok := msg.(*of10.FeaturesReply)
	if !ok {
		panic("unexpected message structure type!")
	}
	r.device = findDevice(reply.DPID)
	r.device.NumBuffers = uint(reply.NumBuffers)
	r.device.NumTables = uint(reply.NumTables)
	r.device.addTransceiver(0, r)

	// XXX: debugging
	r.log.Printf("FeaturesReply: %+v", reply)
	getconfig := of10.NewGetConfigRequest(r.getTransactionID())
	if err := openflow.WriteMessage(r.stream, getconfig); err != nil {
		return err
	}

	return nil
}

func (r *OF10Transceiver) handleGetConfigReply(msg openflow.Message) error {
	reply, ok := msg.(*of10.GetConfigReply)
	if !ok {
		panic("unexpected message structure type!")
	}

	// XXX: debugging
	r.log.Printf("GetConfigReply: %+v", reply)

	return nil
}

func (r *OF10Transceiver) handleMessage(msg openflow.Message) error {
	header := msg.Header()
	if header.Version != r.version {
		return errors.New("unexpected openflow protocol version!")
	}

	switch header.Type {
	case of10.OFPT_ECHO_REQUEST:
		return r.handleEchoRequest(msg)
	case of10.OFPT_ECHO_REPLY:
		return r.handleEchoReply(msg)
	case of10.OFPT_FEATURES_REPLY:
		return r.handleFeaturesReply(msg)
	case of10.OFPT_GET_CONFIG_REPLY:
		return r.handleGetConfigReply(msg)
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

func (r *OF10Transceiver) Run(ctx context.Context) {
	defer r.cleanup()
	r.stream.SetReadTimeout(1 * time.Second)
	r.stream.SetWriteTimeout(5 * time.Second)

	if err := r.sendHello(); err != nil {
		r.log.Printf("Failed to send hello message: %v", err)
		return
	}
	if err := r.sendFeaturesRequest(); err != nil {
		r.log.Printf("Failed to send features_request message: %v", err)
		return
	}
	if err := r.sendSetConfig(of10.OFPC_FRAG_NORMAL, 0xFFFF); err != nil {
		r.log.Printf("Failed to send set_config message: %v", err)
		return
	}
	if err := r.sendBarrierRequest(); err != nil {
		r.log.Printf("Failed to send barrier_request: %v", err)
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
