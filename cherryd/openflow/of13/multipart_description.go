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
	"strings"
)

type DescriptionRequest struct {
	openflow.Message
}

func NewDescriptionRequest(xid uint32) *DescriptionRequest {
	return &DescriptionRequest{
		Message: openflow.NewMessage(openflow.Ver13, OFPT_MULTIPART_REQUEST, xid),
	}
}

func (r *DescriptionRequest) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	// Multipart description request
	binary.BigEndian.PutUint16(v[0:2], OFPMP_DESC)
	// No flags and body
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
	if payload == nil || len(payload) < 1064 {
		return openflow.ErrInvalidPacketLength
	}
	r.Manufacturer = strings.TrimRight(string(payload[8:264]), "\x00")
	r.Hardware = strings.TrimRight(string(payload[264:520]), "\x00")
	r.Software = strings.TrimRight(string(payload[520:776]), "\x00")
	r.Serial = strings.TrimRight(string(payload[776:808]), "\x00")
	r.Description = strings.TrimRight(string(payload[808:1064]), "\x00")

	return nil
}
