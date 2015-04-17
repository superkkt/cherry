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

type Match interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	SetWildcardInPort() error
	SetInPort(port uint32) error
	InPort() (wildcard bool, inport uint32)
	SetWildcardEtherType() error
	SetEtherType(t uint16) error
	EtherType() (wildcard bool, etherType uint16)
	SetWildcardVLANID() error
	SetVLANID(id uint16) error
	VLANID() (wildcard bool, vlanID uint16)
	SetWildcardVLANPriority() error
	SetVLANPriority(p uint8) error
	VLANPriority() (wildcard bool, priority uint8)
	SetWildcardSrcMAC() error
	SetSrcMAC(mac net.HardwareAddr) error
	SrcMAC() (wildcard bool, mac net.HardwareAddr)
	SetWildcardDstMAC() error
	SetDstMAC(mac net.HardwareAddr) error
	DstMAC() (wildcard bool, mac net.HardwareAddr)
	SetWildcardIPProtocol() error
	SetIPProtocol(p uint8) error
	IPProtocol() (wildcard bool, protocol uint8)
	SetSrcIP(ip *net.IPNet) error
	SrcIP() *net.IPNet
	SetDstIP(ip *net.IPNet) error
	DstIP() *net.IPNet
	SetWildcardSrcPort() error
	SetSrcPort(p uint16) error
	SrcPort() (wildcard bool, port uint16)
	SetWildcardDstPort() error
	SetDstPort(p uint16) error
	DstPort() (wildcard bool, port uint16)
}
