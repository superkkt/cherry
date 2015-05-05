/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of10

import (
	"encoding/binary"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type PacketIn struct {
	openflow.Message
	BufferID uint32
	Length   uint16
	InPort   uint16
	Reason   uint8
	Data     []byte
}

func (r *PacketIn) GetBufferID() uint32 {
	return r.BufferID
}

func (r *PacketIn) GetInPort() uint32 {
	return uint32(r.InPort)
}

func (r *PacketIn) GetData() []byte {
	return r.Data
}

func (r *PacketIn) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 10 {
		return openflow.ErrInvalidPacketLength
	}
	r.BufferID = binary.BigEndian.Uint32(payload[0:4])
	r.Length = binary.BigEndian.Uint16(payload[4:6])
	r.InPort = binary.BigEndian.Uint16(payload[6:8])
	r.Reason = payload[8]
	// payload[9] is padding
	if len(payload) >= 10 {
		// TODO: Check data size by comparing with r.Length
		r.Data = payload[10:]
	}

	return nil
}
