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

type ConfigFlag uint16

const (
	FragNormal ConfigFlag = iota
	FragDrop
	FragReasm
	FragMask
)

type Config interface {
	Flags() ConfigFlag
	SetFlags(flags ConfigFlag)
	MissSendLength() uint16
	SetMissSendLength(length uint16)
}

type SetConfig interface {
	Header
	Config
	encoding.BinaryMarshaler
}

type GetConfigRequest interface {
	Header
	encoding.BinaryMarshaler
}

type GetConfigReply interface {
	Header
	Config
	encoding.BinaryUnmarshaler
}
