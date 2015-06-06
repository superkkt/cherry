/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding"
)

type BarrierRequest interface {
	Header
	encoding.BinaryMarshaler
}

type BarrierReply interface {
	Header
	encoding.BinaryUnmarshaler
}
