/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"errors"
)

var (
	ErrInvalidPacketLength     = errors.New("invalid packet length")
	ErrUnsupportedVersion      = errors.New("unsupported protocol version")
	ErrUnsupportedMarshaling   = errors.New("invalid marshaling")
	ErrUnsupportedUnmarshaling = errors.New("invalid unmarshaling")
	ErrUnsupportedMessage      = errors.New("unsupported message type")
)

func IsTimeout(err error) bool {
	type Timeout interface {
		Timeout() bool
	}

	if v, ok := err.(Timeout); ok {
		return v.Timeout()
	}

	return false
}
