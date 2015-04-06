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
	"golang.org/x/net/context"
	"net"
	"sync/atomic"
	"time"
)

type Transceiver interface {
	Run(context.Context)
}

type BaseTransceiver struct {
	stream *openflow.Stream
	log    Logger
	xid    uint32
	device *Device
}

func (r *BaseTransceiver) getTransactionID() uint32 {
	// xid will be started from 1, not 0.
	return atomic.AddUint32(&r.xid, 1)
}

func (r *BaseTransceiver) handleEchoRequest(msg openflow.Message) error {
	m, ok := msg.(*openflow.EchoRequest)
	if !ok {
		panic("unexpected message structure type!")
	}
	header := m.Header()
	reply := openflow.NewEchoReply(header.Version, header.XID, m.Data)
	if err := openflow.WriteMessage(r.stream, reply); err != nil {
		return fmt.Errorf("failed to send echo reply message: %v", err)
	}

	// XXX: debugging
	r.log.Printf("EchoRequest: %+v", m)

	return nil
}

func (r *BaseTransceiver) handleEchoReply(msg openflow.Message) error {
	m, ok := msg.(*openflow.EchoReply)
	if !ok {
		panic("unexpected message structure type!")
	}

	// XXX: debugging
	r.log.Printf("EchoReply: %+v", m)

	return nil
}

func (r *BaseTransceiver) sendEchoRequest() error {
	// TODO: implement this function
	return nil
}

// TODO: Implement to close the connection if we miss several echo replies
func (r *BaseTransceiver) pinger(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(15) * time.Second)
	for {
		select {
		case <-ticker.C:
			if err := r.sendEchoRequest(); err != nil {
				r.log.Printf("failed to send echo request: %v", err)
			}
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func NewTransceiver(conn net.Conn, log Logger) (Transceiver, error) {
	stream := openflow.NewStream(conn)
	stream.SetReadTimeout(5 * time.Second)
	msg, err := openflow.ReadMessage(stream)
	if err != nil {
		return nil, err
	}

	header := msg.Header()
	// The first message should be HELLO
	if header.Type != 0x0 {
		return nil, errors.New("negotiation error: missing HELLO message")
	}

	if header.Version < openflow.Ver13 {
		return NewOF10Transceiver(stream, log), nil
	} else {
		return NewOF13Transceiver(stream, log), nil
	}
}

func addTransceiver(dpid uint64, auxID uint, t Transceiver) *Device {
	v := Pool.Get(dpid)
	if v != nil {
		v.AddTransceiver(auxID, t)
		return v
	}

	v = newDevice(dpid)
	v.AddTransceiver(auxID, t)
	Pool.add(dpid, v)

	return v
}
