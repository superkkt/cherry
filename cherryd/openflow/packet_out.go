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
	Header
	InPort() InPort
	SetInPort(port InPort) error
	Action() Action
	SetAction(action Action) error
	Data() []byte
	SetData(data []byte) error
	encoding.BinaryMarshaler
}
