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
	// SetWildcardInPort sets switch port number as a wildcard
	SetWildcardInPort() error
	// SetInPort sets switch port number
	SetInPort(port InPort) error
	// InPort returns switch port number
	InPort() (wildcard bool, inport InPort)
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
	// SetWildcardSrcPort sets protocol (TCP or UDP) source port number as a wildcard
	SetWildcardSrcPort() error
	// SetSrcPort sets protocol (TCP or UDP) source port number
	SetSrcPort(p uint16) error
	// SrcPort returns protocol (TCP or UDP) source port number
	SrcPort() (wildcard bool, port uint16)
	// SetWildcardDstPort sets protocol (TCP or UDP) destination port number as a wildcard
	SetWildcardDstPort() error
	// SetDstPort sets protocol (TCP or UDP) destination port number
	SetDstPort(p uint16) error
	// DstPort returns protocol (TCP or UDP) destination port number
	DstPort() (wildcard bool, port uint16)
}
