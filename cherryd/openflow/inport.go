/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

type InPort struct {
	port       uint
	controller bool
}

func NewInPort() InPort {
	return InPort{
		controller: true,
	}
}

func (r *InPort) SetPort(port uint) {
	r.controller = false
	r.port = port
}

func (r *InPort) IsController() bool {
	return r.controller
}

func (r *InPort) Port() uint {
	return r.port
}
