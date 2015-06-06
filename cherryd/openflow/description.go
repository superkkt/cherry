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

// Description reqeust
type DescRequest interface {
	Header
	encoding.BinaryMarshaler
}

// Description reply
type DescReply interface {
	Header
	Manufacturer() string
	Hardware() string
	Software() string
	Serial() string
	Description() string
	encoding.BinaryUnmarshaler
}
