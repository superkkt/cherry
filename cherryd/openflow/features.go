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

type FeaturesRequest interface {
	Header
	encoding.BinaryMarshaler
}

type FeaturesReply interface {
	Header
	DPID() uint64
	NumBuffers() uint32
	NumTables() uint8
	Capabilities() uint32
	Actions() uint32
	Ports() []Port
	AuxID() uint8
	encoding.BinaryUnmarshaler
}
