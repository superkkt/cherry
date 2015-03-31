/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding/binary"
	"errors"
	"net"
)

type FlowWildcard struct {
	InPort    bool /* Switch input port. */
	VLANID    bool /* VLAN id. */
	SrcMAC    bool /* Ethernet source address. */
	DstMAC    bool /* Ethernet destination address. */
	EtherType bool /* Ethernet frame type. */
	Protocol  bool /* IP protocol. */
	SrcPort   bool /* TCP/UDP source port. */
	DstPort   bool /* TCP/UDP destination port. */
	// IP source address wildcard bit count. 0 is exact match,
	// 1 ignores the LSB, 2 ignores the 2 least-significant bits, ...,
	// 32 and higher wildcard the entire field.
	SrcIP        uint8
	DstIP        uint8
	VLANPriority bool /* VLAN priority. */
	TOS          bool /* IP ToS (DSCP field, 6 bits). */
}

func newFlowWildcardAll() *FlowWildcard {
	return &FlowWildcard{
		InPort:       true,
		VLANID:       true,
		SrcMAC:       true,
		DstMAC:       true,
		EtherType:    true,
		Protocol:     true,
		SrcPort:      true,
		DstPort:      true,
		SrcIP:        32,
		DstIP:        32,
		VLANPriority: true,
		TOS:          true,
	}
}

func (r *FlowWildcard) MarshalBinary() ([]byte, error) {
	// We only support IPv4 yet
	if r.SrcIP > 32 || r.DstIP > 32 {
		return nil, errors.New("invalid IP address wildcard bit count")
	}

	var v uint32 = 0

	if r.InPort {
		v = v | OFPFW_IN_PORT
	}
	if r.VLANID {
		v = v | OFPFW_DL_VLAN
	}
	if r.SrcMAC {
		v = v | OFPFW_DL_SRC
	}
	if r.DstMAC {
		v = v | OFPFW_DL_DST
	}
	if r.EtherType {
		v = v | OFPFW_DL_TYPE
	}
	if r.Protocol {
		v = v | OFPFW_NW_PROTO
	}
	if r.SrcPort {
		v = v | OFPFW_TP_SRC
	}
	if r.DstPort {
		v = v | OFPFW_TP_DST
	}
	if r.SrcIP > 0 {
		v = v | (uint32(r.SrcIP) << 8)
	}
	if r.DstIP > 0 {
		v = v | (uint32(r.DstIP) << 14)
	}
	if r.VLANPriority {
		v = v | OFPFW_DL_VLAN_PCP
	}
	if r.TOS {
		v = v | OFPFW_NW_TOS
	}

	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data[0:4], v)

	return data, nil
}

func (r *FlowWildcard) UnmarshalBinary(data []byte) error {
	if data == nil || len(data) != 4 {
		return ErrInvalidPacketLength
	}

	w := binary.BigEndian.Uint32(data[0:4])

	if w&OFPFW_IN_PORT != 0 {
		r.InPort = true
	}
	if w&OFPFW_DL_VLAN != 0 {
		r.VLANID = true
	}
	if w&OFPFW_DL_SRC != 0 {
		r.SrcMAC = true
	}
	if w&OFPFW_DL_DST != 0 {
		r.DstMAC = true
	}
	if w&OFPFW_DL_TYPE != 0 {
		r.EtherType = true
	}
	if w&OFPFW_NW_PROTO != 0 {
		r.Protocol = true
	}
	if w&OFPFW_TP_SRC != 0 {
		r.SrcPort = true
	}
	if w&OFPFW_TP_DST != 0 {
		r.DstPort = true
	}
	r.SrcIP = uint8((w & (uint32(0x3F) << 8)) >> 8)
	r.DstIP = uint8((w & (uint32(0x3F) << 14)) >> 14)
	if w&OFPFW_DL_VLAN_PCP != 0 {
		r.VLANPriority = true
	}
	if w&OFPFW_NW_TOS != 0 {
		r.TOS = true
	}

	return nil
}

type FlowMatch struct {
	wildcards    *FlowWildcard
	inPort       uint16
	srcMAC       net.HardwareAddr
	dstMAC       net.HardwareAddr
	vlanID       uint16
	vlanPriority uint8
	etherType    uint16
	tos          uint8
	protocol     uint8
	srcIP        net.IP
	dstIP        net.IP
	srcPort      uint16
	dstPort      uint16
}

// NewFlowMatch() returns a FlowMatch whose fields are all wildcarded
func NewFlowMatch() *FlowMatch {
	srcMAC, err := net.ParseMAC("00:00:00:00:00:00")
	if err != nil {
		panic("Invalid initial MAC address!")
	}
	dstMAC, err := net.ParseMAC("00:00:00:00:00:00")
	if err != nil {
		panic("Invalid initial MAC address!")
	}

	return &FlowMatch{
		wildcards: newFlowWildcardAll(),
		srcMAC:    srcMAC,
		dstMAC:    dstMAC,
		srcIP:     net.ParseIP("0.0.0.0"),
		dstIP:     net.ParseIP("0.0.0.0"),
	}
}

func (r *FlowMatch) SetSrcPort(p uint16) {
	r.srcPort = p
	r.wildcards.SrcPort = false
}

func (r *FlowMatch) GetSrcPort() uint16 {
	return r.srcPort
}

func (r *FlowMatch) SetDstPort(p uint16) {
	r.dstPort = p
	r.wildcards.DstPort = false
}

func (r *FlowMatch) GetDstPort() uint16 {
	return r.dstPort
}

func (r *FlowMatch) SetVLANID(id uint16) {
	r.vlanID = id
	r.wildcards.VLANID = false
}

func (r *FlowMatch) GetVLANID() uint16 {
	return r.vlanID
}

func (r *FlowMatch) SetVLANPriority(p uint8) {
	r.vlanPriority = p
	r.wildcards.VLANPriority = false
}

func (r *FlowMatch) GetVLANPriority() uint8 {
	return r.vlanPriority
}

func (r *FlowMatch) SetTOS(tos uint8) {
	r.tos = tos
	r.wildcards.TOS = false
}

func (r *FlowMatch) GetTOS() uint8 {
	return r.tos
}

func (r *FlowMatch) SetProtocol(p uint8) {
	r.protocol = p
	r.wildcards.Protocol = false
}

func (r *FlowMatch) GetProtocol() uint8 {
	return r.protocol
}

func (r *FlowMatch) GetFlowWildcards() FlowWildcard {
	v := *r.wildcards
	return v
}

func (r *FlowMatch) SetInPort(port uint16) {
	r.inPort = port
	r.wildcards.InPort = false
}

func (r *FlowMatch) GetInPort() uint16 {
	return r.inPort
}

func (r *FlowMatch) SetSrcMAC(mac net.HardwareAddr) {
	r.srcMAC = mac
	r.wildcards.SrcMAC = false
}

func (r *FlowMatch) GetSrcMAC() net.HardwareAddr {
	return r.srcMAC
}

func (r *FlowMatch) SetDstMAC(mac net.HardwareAddr) {
	r.dstMAC = mac
	r.wildcards.DstMAC = false
}

func (r *FlowMatch) GetDstMAC() net.HardwareAddr {
	return r.dstMAC
}

func (r *FlowMatch) SetSrcIP(ip *net.IPNet) {
	r.srcIP = ip.IP

	netmaskBits, _ := ip.Mask.Size()
	if netmaskBits >= 32 {
		r.wildcards.SrcIP = 0
	} else {
		r.wildcards.SrcIP = uint8(32 - netmaskBits)
	}
}

func (r *FlowMatch) GetSrcIP() *net.IPNet {
	return &net.IPNet{
		IP:   r.srcIP,
		Mask: net.CIDRMask(32-int(r.wildcards.SrcIP), 32),
	}
}

func (r *FlowMatch) SetDstIP(ip *net.IPNet) {
	r.dstIP = ip.IP

	netmaskBits, _ := ip.Mask.Size()
	if netmaskBits >= 32 {
		r.wildcards.DstIP = 0
	} else {
		r.wildcards.DstIP = uint8(32 - netmaskBits)
	}
}

func (r *FlowMatch) GetDstIP() *net.IPNet {
	return &net.IPNet{
		IP:   r.dstIP,
		Mask: net.CIDRMask(32-int(r.wildcards.DstIP), 32),
	}
}

func (r *FlowMatch) SetEtherType(t uint16) {
	r.etherType = t
	r.wildcards.EtherType = false
}

func (r *FlowMatch) GetEtherType() uint16 {
	return r.etherType
}

// TODO: other setters and getters for FlowMatch

func (r *FlowMatch) MarshalBinary() ([]byte, error) {
	wildcard, err := r.wildcards.MarshalBinary()
	if err != nil {
		return nil, err
	}

	data := make([]byte, 40)
	copy(data[0:4], wildcard)
	binary.BigEndian.PutUint16(data[4:6], r.inPort)
	copy(data[6:12], r.srcMAC)
	copy(data[12:18], r.dstMAC)
	binary.BigEndian.PutUint16(data[18:20], r.vlanID)
	data[20] = r.vlanPriority
	// data[21] = padding
	binary.BigEndian.PutUint16(data[22:24], r.etherType)
	data[24] = r.tos
	data[25] = r.protocol
	// data[26:28] = padding
	// TODO: Test that big-endian representation for IP is correct
	copy(data[28:32], []byte(r.srcIP.To4()))
	copy(data[32:36], []byte(r.dstIP.To4()))
	binary.BigEndian.PutUint16(data[36:38], r.srcPort)
	binary.BigEndian.PutUint16(data[38:40], r.dstPort)

	return data, nil
}

func (r *FlowMatch) UnmarshalBinary(data []byte) error {
	if len(data) != 40 {
		return ErrInvalidPacketLength
	}

	r.wildcards = &FlowWildcard{}
	if err := r.wildcards.UnmarshalBinary(data[0:4]); err != nil {
		return err
	}
	r.inPort = binary.BigEndian.Uint16(data[4:6])
	r.srcMAC = make(net.HardwareAddr, 6)
	copy(r.srcMAC, data[6:12])
	r.dstMAC = make(net.HardwareAddr, 6)
	copy(r.dstMAC, data[12:18])
	r.vlanID = binary.BigEndian.Uint16(data[18:20])
	r.vlanPriority = data[20]
	r.etherType = binary.BigEndian.Uint16(data[22:24])
	r.tos = data[24]
	r.protocol = data[25]
	// data[26:28] = padding
	// TODO: Test that big-endian representation for IP is correct
	r.srcIP = net.IPv4(data[28], data[29], data[30], data[31])
	r.dstIP = net.IPv4(data[32], data[33], data[34], data[35])
	r.srcPort = binary.BigEndian.Uint16(data[36:38])
	r.dstPort = binary.BigEndian.Uint16(data[38:40])

	return nil
}
