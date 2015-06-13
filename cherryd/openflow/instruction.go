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

type Instruction interface {
	GotoTable(tableID uint8) error
	WriteAction(act Action) error
	ApplyAction(act Action) error
	encoding.BinaryMarshaler
}
