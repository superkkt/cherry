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
	DstIP() *net.IPNet
	DstMAC() (wildcard bool, mac net.HardwareAddr)
	// DstPort returns protocol (TCP or UDP) destination port number
	DstPort() (wildcard bool, port uint16)
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	Error() error
	EtherType() (wildcard bool, etherType uint16)
	// InPort returns switch port number
	InPort() (wildcard bool, inport InPort)
	IPProtocol() (wildcard bool, protocol uint8)
	SetDstIP(ip *net.IPNet)
	SetDstMAC(mac net.HardwareAddr)
	// SetDstPort sets protocol (TCP or UDP) destination port number
	SetDstPort(p uint16)
	SetEtherType(t uint16)
	// SetInPort sets switch port number
	SetInPort(port InPort)
	SetIPProtocol(p uint8)
	SetSrcIP(ip *net.IPNet)
	SetSrcMAC(mac net.HardwareAddr)
	// SetSrcPort sets protocol (TCP or UDP) source port number
	SetSrcPort(p uint16)
	SetVLANID(id uint16)
	SetVLANPriority(p uint8)
	SetWildcardEtherType()
	SetWildcardDstMAC()
	// SetWildcardDstPort sets protocol (TCP or UDP) destination port number as a wildcard
	SetWildcardDstPort()
	SetWildcardSrcMAC()
	// SetWildcardSrcPort sets protocol (TCP or UDP) source port number as a wildcard
	SetWildcardSrcPort()
	// SetWildcardInPort sets switch port number as a wildcard
	SetWildcardInPort()
	SetWildcardIPProtocol()
	SetWildcardVLANID()
	SetWildcardVLANPriority()
	SrcIP() *net.IPNet
	SrcMAC() (wildcard bool, mac net.HardwareAddr)
	// SrcPort returns protocol (TCP or UDP) source port number
	SrcPort() (wildcard bool, port uint16)
	VLANID() (wildcard bool, vlanID uint16)
	VLANPriority() (wildcard bool, priority uint8)
}
