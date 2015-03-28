/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

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

func (r *Manager) handleHelloMessage(msg *openflow.HelloMessage) error {
	// We only support OF 1.0
	if msg.Version < 0x01 {
		fmt.Errorf("unsupported OpenFlow protocol version: 0x%X", msg.Version)
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

	return nil
}

func (r *Manager) handleEchoRequestMessage(msg *openflow.EchoRequestMessage) error {
	r.log.Printf("%+v", msg)
	return nil
}

func (r *Manager) handleEchoReplyMessage(msg *openflow.EchoReplyMessage) error {
	r.log.Printf("%+v", msg)
	return nil
}

func (r *Manager) Run(ctx context.Context, conn net.Conn) {
	socket := socket.NewConn(conn, 65536) // 65536 bytes are max size of a OpenFlow packet

	config := openflow.Config{
		Log:          r.log,
		Socket:       socket,
		ReadTimeout:  socketTimeout,
		WriteTimeout: socketTimeout,
		Handlers: openflow.MessageHandler{
			HelloMessage:         r.handleHelloMessage,
			ErrorMessage:         r.handleErrorMessage,
			FeaturesReplyMessage: r.handleFeaturesReplyMessage,
			EchoRequestMessage:   r.handleEchoRequestMessage,
			EchoReplyMessage:     r.handleEchoReplyMessage,
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
