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

type PortReason uint8

const (
	PortAdded PortReason = iota
	PortDeleted
	PortModified
)

type PortStatus interface {
	Header
	Reason() PortReason
	Port() Port
	encoding.BinaryUnmarshaler
}
