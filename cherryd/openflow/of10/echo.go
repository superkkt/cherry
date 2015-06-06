/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of10

import (
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

func NewEchoRequest(xid uint32) openflow.EchoRequest {
	return &openflow.BaseEcho{
		Message: openflow.NewMessage(openflow.OF10_VERSION, OFPT_ECHO_REQUEST, xid),
	}
}

func NewEchoReply(xid uint32) openflow.EchoReply {
	return &openflow.BaseEcho{
		Message: openflow.NewMessage(openflow.OF10_VERSION, OFPT_ECHO_REPLY, xid),
	}
}
