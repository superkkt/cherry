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
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"golang.org/x/net/context"
	"net"
	"time"
)

const (
	socketTimeout = 5 * time.Second
)

type Manager struct {
	log        log.Logger
	ofp        *openflow.Protocol
	negotiated bool // Negotiation of protocol version
}

func NewManager(log log.Logger) *Manager {
	return &Manager{
		log: log,
	}
}

func isTimeout(err error) bool {
	type Timeout interface {
		Timeout() bool
	}

	if v, ok := err.(Timeout); ok {
		return v.Timeout()
	}

	return false
}

func (r *Manager) handleHelloMessage(msg *openflow.HelloMessage) error {
	// We only support OF 1.0
	if msg.Version < 0x01 {
		// TODO: send error code
		return fmt.Errorf("unsupported OpenFlow protocol version: %v", msg.Version)
	}
	if err := r.ofp.SendFeaturesRequestMessage(); err != nil {
		return err
	}
	r.negotiated = true

	return nil
}

func (r *Manager) handleMessage(msg interface{}) error {
	switch v := msg.(type) {
	case *openflow.HelloMessage:
		return r.handleHelloMessage(v)

	default:
		panic("unsupported message type!")
	}

	return nil
}

func (r *Manager) Run(ctx context.Context, conn net.Conn) {
	socket := socket.NewConn(conn, 65536) // Type of length in OpenFlow packet header is uint16
	defer socket.Close()

	r.ofp = openflow.NewProtocol(socket, socketTimeout, socketTimeout)
	if err := r.ofp.SendHelloMessage(); err != nil {
		r.log.Err(fmt.Sprintf("SendHelloMessage: %v", err.Error()))
		return
	}

	// Reader goroutine
	receivedMsg := make(chan interface{})
	go func() {
		for {
			msg, err := r.ofp.ReadMessage()
			if err != nil {
				switch {
				case isTimeout(err):
					// Ignore timeout error
					continue

				case err == openflow.ErrUnsupportedMsgType:
					r.log.Err(fmt.Sprintf("ReadMessage: %v", err.Error()))
					continue

				default:
					r.log.Err(fmt.Sprintf("ReadMessage: %v", err.Error()))
					close(receivedMsg)
					return
				}
			}
			receivedMsg <- msg
		}
	}()

	for {
		select {
		case msg, ok := <-receivedMsg:
			if !ok {
				return
			}
			if err := r.handleMessage(msg); err != nil {
				r.log.Err(fmt.Sprintf("handleMessage: %v", err.Error()))
				return
			}

		// TODO: add this manager to the device pool with DPID

		case <-ctx.Done():
			return
		}
	}
}
