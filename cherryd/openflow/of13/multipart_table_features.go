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

type TableFeaturesRequest struct {
	openflow.Message
}

func NewTableFeaturesRequest(xid uint32) *TableFeaturesRequest {
	return &TableFeaturesRequest{
		Message: openflow.NewMessage(openflow.Ver13, OFPT_MULTIPART_REQUEST, xid),
	}
}

func (r *TableFeaturesRequest) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	// Table features request
	binary.BigEndian.PutUint16(v[0:2], OFPMP_TABLE_FEATURES)
	// No flags and body
	r.SetPayload(v)

	return r.Message.MarshalBinary()
}

// TODO: Implement TableFeaturesReply
