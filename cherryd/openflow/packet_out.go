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

type PacketOut interface {
	Action() Action
	Data() []byte
	encoding.BinaryMarshaler
	Error() error
	Header
	InPort() InPort
	SetAction(action Action)
	SetData(data []byte)
	SetInPort(port InPort)
}
