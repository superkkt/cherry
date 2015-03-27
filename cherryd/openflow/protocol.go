/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding"
	"encoding/binary"
	"errors"
	"git.sds.co.kr/bosomi.git/socket"
	"time"
)

const (
	socketTimeout = 5 * time.Second
)

var (
	ErrUnsupportedMsgType = errors.New("unsupported message type")
)

type Protocol struct {
	socket       *socket.Conn
	xid          uint32 // Transaction ID associated with OpenFlow packets
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func NewProtocol(s *socket.Conn, rt, wt time.Duration) *Protocol {
	return &Protocol{
		socket:       s,
		xid:          0,
		readTimeout:  rt,
		writeTimeout: wt,
	}
}

func (r *Protocol) Close() error {
	return r.socket.Close()
}

func (r *Protocol) send(data encoding.BinaryMarshaler) error {
	v, err := data.MarshalBinary()
	if err != nil {
		return err
	}

	if r.writeTimeout > 0 {
		r.socket.SetDeadline(time.Now().Add(r.writeTimeout))
		defer r.socket.SetDeadline(time.Time{})
	}
	_, err = r.socket.Write(v)
	if err != nil {
		return err
	}

	return nil
}

func (r *Protocol) getTransactionID() uint32 {
	v := r.xid
	r.xid++
	return v
}

func (r *Protocol) SendHelloMessage() error {
	msg := &HelloMessage{
		Header{
			Version: 0x01, // OF1.0
			Type:    OFPT_HELLO,
			Length:  8,
			Xid:     r.getTransactionID(),
		},
	}

	return r.send(msg)
}

func (r *Protocol) SendFeaturesRequestMessage() error {
	msg := &FeaturesRequestMessage{
		Header{
			Version: 0x01, // OF1.0
			Type:    OFPT_FEATURES_REQUEST,
			Length:  8,
			Xid:     r.getTransactionID(),
		},
	}

	return r.send(msg)
}

func (r *Protocol) SendNegotiationFailedMessage() error {
	msg := &ErrorMessage{
		Header: Header{
			Version: 0x01, // OF1.0
			Type:    OFPT_ERROR,
			Length:  0,
			Xid:     r.getTransactionID(),
		},
		Type: OFPET_HELLO_FAILED,
		Code: OFPHFC_INCOMPATIBLE,
		Data: []byte("Sorry. We only support OpenFlow 1.0."),
	}
	msg.Length = uint16(12 + len(msg.Data))

	return r.send(msg)
}

func parsePacket(packet []byte) (interface{}, error) {
	var msg encoding.BinaryUnmarshaler

	switch packet[1] {
	case OFPT_HELLO:
		msg = &HelloMessage{}
	case OFPT_ERROR:
		msg = &ErrorMessage{}
	case OFPT_FEATURES_REPLY:
		msg = &FeaturesReplyMessage{}
	default:
		return nil, ErrUnsupportedMsgType
	}

	if err := msg.UnmarshalBinary(packet); err != nil {
		return nil, err
	}
	return msg, nil
}

func (r *Protocol) ReadMessage() (interface{}, error) {
	if r.readTimeout > 0 {
		r.socket.SetDeadline(time.Now().Add(r.readTimeout))
		defer r.socket.SetDeadline(time.Time{})
	}

	header, err := r.socket.Peek(8) // peek ofp_header
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint16(header[2:4])
	if length < 8 {
		return nil, errors.New("invalid packet length")
	}

	packet, err := r.socket.ReadN(int(length))
	if err != nil {
		return nil, err
	}

	return parsePacket(packet)
}
