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

type FlowRemoved interface {
	Header
	Cookie() uint64
	Priority() uint16
	Reason() uint8
	TableID() uint8
	DurationSec() uint32
	DurationNanoSec() uint32
	IdleTimeout() uint16
	HardTimeout() uint16
	PacketCount() uint64
	ByteCount() uint64
	Match() Match
	encoding.BinaryUnmarshaler
}
