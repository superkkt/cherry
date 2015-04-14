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

type DescriptionRequest struct {
	openflow.Message
}

func NewDescriptionRequest(xid uint32) *DescriptionRequest {
	return &DescriptionRequest{
		Message: openflow.NewMessage(openflow.Ver10, OFPT_STATS_REQUEST, xid),
	}
}

func (r *DescriptionRequest) MarshalBinary() ([]byte, error) {
	v := make([]byte, 4)
	binary.BigEndian.PutUint16(v[0:2], OFPST_DESC)
	// v[2:4] is flags, but not yet defined
	r.SetPayload(v)

	return r.Message.MarshalBinary()
}

type DescriptionReply struct {
	openflow.Message
	Manufacturer string
	Hardware     string
	Software     string
	Serial       string
	Description  string
}

func (r *DescriptionReply) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 1060 {
		return openflow.ErrInvalidPacketLength
	}
	// payload[0:4] is type and flag of ofp_stats_reply
	r.Manufacturer = strings.TrimRight(string(payload[4:260]), "\x00")
	r.Hardware = strings.TrimRight(string(payload[260:516]), "\x00")
	r.Software = strings.TrimRight(string(payload[516:772]), "\x00")
	r.Serial = strings.TrimRight(string(payload[772:804]), "\x00")
	r.Description = strings.TrimRight(string(payload[804:1060]), "\x00")

	return nil
}
