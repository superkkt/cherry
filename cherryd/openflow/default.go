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

var (
	ZeroMAC net.HardwareAddr
	ZeroIP  net.IP
)

func init() {
	mac, err := net.ParseMAC("00:00:00:00:00:00")
	if err != nil {
		panic("Invalid initial MAC address!")
	}
	ZeroMAC = mac
	ZeroIP = net.ParseIP("0.0.0.0")
}
