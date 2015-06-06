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

type PacketIn interface {
	Header
	BufferID() uint32
	Length() uint16
	InPort() uint32
	TableID() uint8
	Reason() uint8
	Cookie() uint64
	Data() []byte
	encoding.BinaryUnmarshaler
}
