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

func NewHello(xid uint32) openflow.Hello {
	return &openflow.BaseHello{
		Message: openflow.NewMessage(openflow.OF13_VERSION, OFPT_HELLO, xid),
	}
}
