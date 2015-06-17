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
	ApplyAction(act Action)
	encoding.BinaryMarshaler
	Error() error
	GotoTable(tableID uint8)
	WriteAction(act Action)
}
