/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

import (
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type FeaturesRequest struct {
	header openflow.Header
}

func NewFeaturesRequest(xid uint32) *FeaturesRequest {
	return &FeaturesRequest{
		header: openflow.Header{
			Version: openflow.Ver13,
			Type:    OFPT_FEATURES_REQUEST,
			Length:  8,
			XID:     xid,
		},
	}
}

func (r *FeaturesRequest) Header() openflow.Header {
	return r.header
}

func (r *FeaturesRequest) MarshalBinary() ([]byte, error) {
	return r.header.MarshalBinary()
}

func (r *FeaturesRequest) UnmarshalBinary(data []byte) error {
	return openflow.ErrUnsupportedUnmarshaling
}
