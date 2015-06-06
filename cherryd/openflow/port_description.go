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

type PortDescRequest interface {
	Header
	encoding.BinaryMarshaler
}

type PortDescReply interface {
	Header
	Ports() []Port
	encoding.BinaryUnmarshaler
}
