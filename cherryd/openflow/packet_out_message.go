/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding/binary"
	"errors"
)

type PacketOutMessage struct {
	Header
	BufferID uint32
	InPort   PortNumber
	Actions  []FlowAction
	Data     []byte
}

func (r *PacketOutMessage) MarshalBinary() ([]byte, error) {
	actions := make([]byte, 0)
	for _, act := range r.Actions {
		buf, err := act.MarshalBinary()
		if err != nil {
			return nil, err
		}
		actions = append(actions, buf...)
	}

	var dataLength uint16 = 0
	if r.Data != nil {
		if len(r.Data) > 65535-16 {
			return nil, errors.New("too long packet data in a packet-out message")
		}
		dataLength = uint16(len(r.Data))
	}
	if len(actions) > int(65535-16-dataLength) {
		return nil, errors.New("too many packet-out actions")
	}

	r.Header.Length = 16 + uint16(len(actions)) + dataLength
	header, err := r.Header.MarshalBinary()
	if err != nil {
		return nil, err
	}

	v := make([]byte, r.Header.Length)
	copy(v[0:8], header)
	// We don't support buffer ID
	r.BufferID = 0xFFFFFFFF
	binary.BigEndian.PutUint32(v[8:12], r.BufferID)
	binary.BigEndian.PutUint16(v[12:14], uint16(r.InPort))
	binary.BigEndian.PutUint16(v[14:16], uint16(len(actions)))
	copy(v[16:16+len(actions)], actions)
	if dataLength > 0 {
		copy(v[16+len(actions):], r.Data)
	}

	return v, nil
}
