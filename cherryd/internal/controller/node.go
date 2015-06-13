/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package controller

import (
	"net"
)

type Node struct {
	port *Port
	mac  net.HardwareAddr
}

func NewNode(p *Port, mac net.HardwareAddr) *Node {
	return &Node{
		port: p,
		mac:  mac,
	}
}

func (r *Node) Port() *Port {
	return r.port
}

func (r *Node) MAC() net.HardwareAddr {
	return r.mac
}
