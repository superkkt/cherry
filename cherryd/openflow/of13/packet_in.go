/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

import (
	"encoding/binary"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type PacketIn struct {
	openflow.Message
	BufferID uint32
	Length   uint16
	InPort   uint32
	TableID  uint8
	Reason   uint8
	Cookie   uint64
	Data     []byte
}

func (r *PacketIn) GetBufferID() uint32 {
	return r.BufferID
}

func (r *PacketIn) GetInPort() uint32 {
	return r.InPort
}

func (r *PacketIn) GetData() []byte {
	return r.Data
}

func (r *PacketIn) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 24 {
		return openflow.ErrInvalidPacketLength
	}
	r.BufferID = binary.BigEndian.Uint32(payload[0:4])
	r.Length = binary.BigEndian.Uint16(payload[4:6])
	r.Reason = payload[6]
	r.TableID = payload[7]
	r.Cookie = binary.BigEndian.Uint64(payload[8:16])

	match := NewMatch()
	if err := match.UnmarshalBinary(payload[16:]); err != nil {
		return err
	}
	_, inport := match.InPort()
	r.InPort = uint32(inport)

	matchLength := binary.BigEndian.Uint16(payload[18:20])
	// Calculate padding length
	rem := matchLength % 8
	if rem > 0 {
		matchLength += 8 - rem
	}

	dataOffset := 16 + matchLength + 2 // +2 is padding
	if len(payload) >= int(dataOffset) {
		// TODO: Check data size by comparing with r.Length
		r.Data = payload[dataOffset:]
	}

	return nil
}
