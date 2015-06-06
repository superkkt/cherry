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

type FeaturesRequest struct {
	openflow.Message
}

func NewFeaturesRequest(xid uint32) openflow.FeaturesRequest {
	return &FeaturesRequest{
		Message: openflow.NewMessage(openflow.OF13_VERSION, OFPT_FEATURES_REQUEST, xid),
	}
}

func (r *FeaturesRequest) MarshalBinary() ([]byte, error) {
	return r.Message.MarshalBinary()
}

type FeaturesReply struct {
	openflow.Message
	dpid         uint64
	numBuffers   uint32
	numTables    uint8
	auxID        uint8
	capabilities uint32
}

func (r FeaturesReply) DPID() uint64 {
	return r.dpid
}

func (r FeaturesReply) NumBuffers() uint32 {
	return r.numBuffers
}

func (r FeaturesReply) NumTables() uint8 {
	return r.numTables
}

func (r FeaturesReply) Capabilities() uint32 {
	return r.capabilities
}

func (r FeaturesReply) Actions() uint32 {
	// OpenFlow 1.3 does not have actions
	return 0
}

func (r FeaturesReply) Ports() []openflow.Port {
	// OpenFlow 1.3 does not have port
	return nil
}

func (r FeaturesReply) AuxID() uint8 {
	return r.auxID
}

func (r *FeaturesReply) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 20 {
		return openflow.ErrInvalidPacketLength
	}
	r.dpid = binary.BigEndian.Uint64(payload[0:8])
	r.numBuffers = binary.BigEndian.Uint32(payload[8:12])
	r.numTables = payload[12]
	r.auxID = payload[13]
	r.capabilities = binary.BigEndian.Uint32(payload[16:20])

	return nil
}
