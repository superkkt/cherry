/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding"
	"net"
)

type Action interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	SetOutPort(port OutPort) error
	OutPort() []OutPort
	SetSrcMAC(mac net.HardwareAddr) error
	SrcMAC() (ok bool, mac net.HardwareAddr)
	SetDstMAC(mac net.HardwareAddr) error
	DstMAC() (ok bool, mac net.HardwareAddr)
}

type BaseAction struct {
	output map[OutPort]interface{}
	srcMAC *net.HardwareAddr
	dstMAC *net.HardwareAddr
}

func NewBaseAction() *BaseAction {
	return &BaseAction{
		output: make(map[OutPort]interface{}),
	}
}

func (r *BaseAction) SetOutPort(port OutPort) error {
	r.output[port] = nil
	return nil
}

func (r *BaseAction) OutPort() []OutPort {
	ports := make([]OutPort, 0)
	for v, _ := range r.output {
		ports = append(ports, v)
	}

	return ports
}

func (r *BaseAction) SetSrcMAC(mac net.HardwareAddr) error {
	if mac == nil || len(mac) < 6 {
		return ErrInvalidMACAddress
	}

	r.srcMAC = &mac
	return nil
}

func (r *BaseAction) SrcMAC() (ok bool, mac net.HardwareAddr) {
	if r.srcMAC == nil {
		return false, ZeroMAC
	}

	return true, *r.srcMAC
}

func (r *BaseAction) SetDstMAC(mac net.HardwareAddr) error {
	if mac == nil || len(mac) < 6 {
		return ErrInvalidMACAddress
	}

	r.dstMAC = &mac
	return nil
}

func (r *BaseAction) DstMAC() (ok bool, mac net.HardwareAddr) {
	if r.dstMAC == nil {
		return false, ZeroMAC
	}

	return true, *r.dstMAC
}
