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
	"strings"
)

type DescRequest struct {
	openflow.Message
}

func NewDescRequest(xid uint32) openflow.DescRequest {
	return &DescRequest{
		Message: openflow.NewMessage(openflow.OF10_VERSION, OFPT_STATS_REQUEST, xid),
	}
}

func (r *DescRequest) MarshalBinary() ([]byte, error) {
	v := make([]byte, 4)
	binary.BigEndian.PutUint16(v[0:2], OFPST_DESC)
	// v[2:4] is flags, but not yet defined
	r.SetPayload(v)

	return r.Message.MarshalBinary()
}

type DescReply struct {
	openflow.Message
	manufacturer string
	hardware     string
	software     string
	serial       string
	description  string
}

func (r DescReply) Manufacturer() string {
	return r.manufacturer
}

func (r DescReply) Hardware() string {
	return r.hardware
}

func (r DescReply) Software() string {
	return r.software
}

func (r DescReply) Serial() string {
	return r.serial
}

func (r DescReply) Description() string {
	return r.description
}

func (r *DescReply) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 1060 {
		return openflow.ErrInvalidPacketLength
	}
	// payload[0:4] is type and flag of ofp_stats_reply
	r.manufacturer = strings.TrimRight(string(payload[4:260]), "\x00")
	r.hardware = strings.TrimRight(string(payload[260:516]), "\x00")
	r.software = strings.TrimRight(string(payload[516:772]), "\x00")
	r.serial = strings.TrimRight(string(payload[772:804]), "\x00")
	r.description = strings.TrimRight(string(payload[804:1060]), "\x00")

	return nil
}
