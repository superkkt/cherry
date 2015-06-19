/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package app

import (
	"git.sds.co.kr/cherry.git/cherryd/internal/network"
)

type Processor interface {
	network.EventListener
	// Name returns the application name that is globally unique
	Name() string
	Next() (next Processor, ok bool)
	SetNext(Processor)
}
