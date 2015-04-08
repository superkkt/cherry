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
	header openflow.Header
}

func NewDescriptionRequest(xid uint32) *DescriptionRequest {
	return &DescriptionRequest{
		header: openflow.Header{
			Version: openflow.Ver13,
			Type:    OFPT_MULTIPART_REQUEST,
			XID:     xid,
		},
	}
}

func (r *DescriptionRequest) Header() openflow.Header {
	return r.header
}

func (r *DescriptionRequest) MarshalBinary() ([]byte, error) {
	r.header.Length = 16
	header, err := r.header.MarshalBinary()
	if err != nil {
		return nil, err
	}

	v := make([]byte, r.header.Length)
	copy(v[0:8], header)
	binary.BigEndian.PutUint16(v[8:10], OFPMP_DESC)

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
	if r.header.Length < 1072 || len(data) < int(r.header.Length) {
		return openflow.ErrInvalidPacketLength
	}

	r.Manufacturer = strings.TrimRight(string(data[16:272]), "\x00")
	r.Hardware = strings.TrimRight(string(data[272:528]), "\x00")
	r.Software = strings.TrimRight(string(data[528:784]), "\x00")
	r.Serial = strings.TrimRight(string(data[784:816]), "\x00")
	r.Description = strings.TrimRight(string(data[816:1072]), "\x00")

	return nil
}
