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

type FlowMatch interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	SetWildcardInPort() error
	SetInPort(port uint) error
	GetInPort() (wildcard bool, inport uint)
	SetWildcardEtherType() error
	SetEtherType(t uint16) error
	GetEtherType() (wildcard bool, etherType uint16)
	SetWildcardVLANID() error
	SetVLANID(id uint16) error
	GetVLANID() (wildcard bool, vlanID uint16)
	SetWildcardVLANPriority() error
	SetVLANPriority(p uint8) error
	GetVLANPriority() (wildcard bool, priority uint8)
	SetWildcardSrcMAC() error
	SetSrcMAC(mac net.HardwareAddr) error
	GetSrcMAC() (wildcard bool, mac net.HardwareAddr)
	SetWildcardDstMAC() error
	SetDstMAC(mac net.HardwareAddr) error
	GetDstMAC() (wildcard bool, mac net.HardwareAddr)
	SetWildcardIPProtocol() error
	SetIPProtocol(p uint8) error
	GetIPProtocol() (wildcard bool, protocol uint8)
	SetSrcIP(ip *net.IPNet) error
	GetSrcIP() *net.IPNet
	SetDstIP(ip *net.IPNet) error
	GetDstIP() *net.IPNet
	SetWildcardSrcPort() error
	SetSrcPort(p uint16) error
	GetSrcPort() (wildcard bool, port uint16)
	SetWildcardDstPort() error
	SetDstPort(p uint16) error
	GetDstPort() (wildcard bool, port uint16)
}
