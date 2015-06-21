/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service 
 * Kitae Kim <superkkt@sds.co.kr>
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along
 * with this program; if not, write to the Free Software Foundation, Inc.,
 * 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
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
