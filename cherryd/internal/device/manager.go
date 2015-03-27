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
	"git.sds.co.kr/bosomi.git/socket"
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"golang.org/x/net/context"
	"io"
	"net"
	"time"
)

const (
	socketTimeout = 5 * time.Second
)

var (
	ErrNoNegotiated = errors.New("no negotiated session")
)

type Manager struct {
	log        log.Logger
	ofp        *openflow.Protocol
	negotiated bool // Negotiation of protocol version
	dpid       string
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
		r.ofp.SendNegotiationFailedMessage()
		return fmt.Errorf("unsupported OpenFlow protocol version: %v", msg.Version)
	}
	if err := r.ofp.SendFeaturesRequestMessage(); err != nil {
		return err
	}
	r.negotiated = true

	return nil
}

func (r *Manager) handleErrorMessage(msg *openflow.ErrorMessage) error {
	e := fmt.Sprintf("error from a device: dpid=%v, type=%v, code=%v, data=%v",
		r.dpid, msg.Type, msg.Code, msg.Data)
	r.log.Err(e)
	return nil
}

func (r *Manager) handleFeaturesReplyMessage(msg *openflow.FeaturesReplyMessage) error {
	if r.negotiated == false {
		return ErrNoNegotiated
	}

	r.log.Debug(fmt.Sprintf("%+v", msg))
	// TODO
	return nil
}

func (r *Manager) handleMessage(msg interface{}) error {
	switch v := msg.(type) {
	case *openflow.HelloMessage:
		return r.handleHelloMessage(v)

	case *openflow.ErrorMessage:
		return r.handleErrorMessage(v)

	case *openflow.FeaturesReplyMessage:
		return r.handleFeaturesReplyMessage(v)

	// TODO: set r.dpid

	default:
		panic("unsupported message type!")
	}

	return nil
}

func (r *Manager) Run(ctx context.Context, conn net.Conn) {
	socket := socket.NewConn(conn, 65536) // Type of length in OpenFlow packet header is uint16
	r.ofp = openflow.NewProtocol(socket, socketTimeout, socketTimeout)
	defer r.ofp.Close()

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
				case err == io.EOF:
					r.log.Debug("EOF received")
					close(receivedMsg)
					return

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
				// Reader goroutine will be finished after it detects socket disconnection
				return
			}

		// TODO: add this manager to the device pool with DPID

		case <-ctx.Done():
			return
		}
	}
}
