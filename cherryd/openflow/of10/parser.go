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
)

func init() {
	openflow.RegisterParser(openflow.Ver10, ParseMessage)
}

func ParseMessage(data []byte) (openflow.Message, error) {
	header := openflow.Header{}
	if err := header.UnmarshalBinary(data); err != nil {
		return nil, err
	}

	var msg openflow.Message

	switch header.Type {
	case OFPT_FEATURES_REPLY:
		msg = new(FeaturesReply)
	case OFPT_GET_CONFIG_REPLY:
		msg = new(GetConfigReply)
	case OFPT_STATS_REPLY:
		switch binary.BigEndian.Uint16(data[8:10]) {
		case OFPST_DESC:
			msg = new(DescriptionReply)
		default:
			return nil, openflow.ErrUnsupportedMessage
		}
	default:
		return nil, openflow.ErrUnsupportedMessage
	}

	if err := msg.UnmarshalBinary(data); err != nil {
		return nil, err
	}

	return msg, nil
}
