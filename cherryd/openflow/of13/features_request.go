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
	openflow.Message
}

func NewFeaturesRequest(xid uint32) *FeaturesRequest {
	return &FeaturesRequest{
		Message: openflow.NewMessage(openflow.Ver13, OFPT_FEATURES_REQUEST, xid),
	}
}

func (r *FeaturesRequest) MarshalBinary() ([]byte, error) {
	return r.Message.MarshalBinary()
}
