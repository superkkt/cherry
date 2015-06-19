/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding"
	"fmt"
	"net"
)

type Action interface {
	DstMAC() (ok bool, mac net.HardwareAddr)
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	// Error() returns last error message
	Error() error
	OutPort() []OutPort
	SetDstMAC(mac net.HardwareAddr)
	SetOutPort(port OutPort)
	SetSrcMAC(mac net.HardwareAddr)
	SrcMAC() (ok bool, mac net.HardwareAddr)
}

type BaseAction struct {
	err    error
	output map[OutPort]interface{}
	srcMAC *net.HardwareAddr
	dstMAC *net.HardwareAddr
}

func NewBaseAction() *BaseAction {
	return &BaseAction{
		output: make(map[OutPort]interface{}),
	}
}

func (r *BaseAction) SetOutPort(port OutPort) {
	r.output[port] = nil
}

func (r *BaseAction) OutPort() []OutPort {
	ports := make([]OutPort, 0)
	for v, _ := range r.output {
		ports = append(ports, v)
	}

	return ports
}

func (r *BaseAction) SetSrcMAC(mac net.HardwareAddr) {
	if mac == nil || len(mac) < 6 {
		r.err = fmt.Errorf("SetSrcMAC: %v", ErrInvalidMACAddress)
		return
	}

	r.srcMAC = &mac
}

func (r *BaseAction) SrcMAC() (ok bool, mac net.HardwareAddr) {
	if r.srcMAC == nil {
		return false, net.HardwareAddr([]byte{0, 0, 0, 0, 0, 0})
	}

	return true, *r.srcMAC
}

func (r *BaseAction) SetDstMAC(mac net.HardwareAddr) {
	if mac == nil || len(mac) < 6 {
		r.err = fmt.Errorf("SetDstMAC: %v", ErrInvalidMACAddress)
		return
	}

	r.dstMAC = &mac
}

func (r *BaseAction) DstMAC() (ok bool, mac net.HardwareAddr) {
	if r.dstMAC == nil {
		return false, net.HardwareAddr([]byte{0, 0, 0, 0, 0, 0})
	}

	return true, *r.dstMAC
}

func (r *BaseAction) Error() error {
	return r.err
}
