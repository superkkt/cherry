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
	header openflow.Header
}

func NewDescriptionRequest(xid uint32) *DescriptionRequest {
	return &DescriptionRequest{
		header: openflow.Header{
			Version: openflow.Ver10,
			Type:    OFPT_STATS_REQUEST,
			XID:     xid,
		},
	}
}

func (r *DescriptionRequest) Header() openflow.Header {
	return r.header
}

func (r *DescriptionRequest) MarshalBinary() ([]byte, error) {
	r.header.Length = 12
	header, err := r.header.MarshalBinary()
	if err != nil {
		return nil, err
	}

	v := make([]byte, r.header.Length)
	copy(v[0:8], header)
	binary.BigEndian.PutUint16(v[8:10], OFPST_DESC)

	return v, nil
}

func (r *DescriptionRequest) UnmarshalBinary(data []byte) error {
	return openflow.ErrUnsupportedUnmarshaling
}

type DescriptionReply struct {
	header       openflow.Header
	Manufacturer string
	Hardware     string
	Software     string
	Serial       string
	Description  string
}

func (r *DescriptionReply) Header() openflow.Header {
	return r.header
}

func (r *DescriptionReply) MarshalBinary() ([]byte, error) {
	return nil, openflow.ErrUnsupportedMarshaling
}

func (r *DescriptionReply) UnmarshalBinary(data []byte) error {
	if err := r.header.UnmarshalBinary(data); err != nil {
		return err
	}
	if r.header.Length < 1068 || len(data) < int(r.header.Length) {
		return openflow.ErrInvalidPacketLength
	}

	r.Manufacturer = strings.TrimRight(string(data[12:268]), "\x00")
	r.Hardware = strings.TrimRight(string(data[268:524]), "\x00")
	r.Software = strings.TrimRight(string(data[524:780]), "\x00")
	r.Serial = strings.TrimRight(string(data[780:812]), "\x00")
	r.Description = strings.TrimRight(string(data[812:1068]), "\x00")

	return nil
}
