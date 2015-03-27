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

type PhysicalPort struct {
	Number uint16
	MAC    net.HardwareAddr
	Name   string
	// Bitmap of OFPPC_* flags
	config uint32
	// Bitmap of OFPPS_* flags
	state uint32
	//
	//  Bitmaps of OFPPF_* that describe features. All bits zeroed if unsupported or unavailable.
	//
	current    uint32
	advertised uint32
	supported  uint32
	peer       uint32
}

// Whether it is administratively down
func (r *PhysicalPort) IsPortDown() bool {
	if r.config&OFPPC_PORT_DOWN != 0 {
		return true
	}

	return false
}

// Whether physical link is present
func (r *PhysicalPort) IsLinkDown() bool {
	if r.state&OFPPS_LINK_DOWN != 0 {
		return true
	}

	return false
}

// TODO: Is* functions for current, advertised, supported, and peer
