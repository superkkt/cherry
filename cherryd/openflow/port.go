/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"net"
)

type Port interface {
	Number() uint
	MAC() net.HardwareAddr
	Name() string
	IsPortDown() bool // Is the port Administratively down?
	IsLinkDown() bool // Is a physical link on the port down?
}
