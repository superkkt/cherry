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

func marshalFlowWildcard(w *FlowWildcard) (uint32, error) {
	// We only support IPv4 yet
	if w.SrcIP > 32 || w.DstIP > 32 {
		return 0, errors.New("invalid IP address wildcard bit count")
	}

	var v uint32 = 0

	if w.InPort {
		v = v | OFPFW_IN_PORT
	}
	if w.VLANID {
		v = v | OFPFW_DL_VLAN
	}
	if w.SrcMAC {
		v = v | OFPFW_DL_SRC
	}
	if w.DstMAC {
		v = v | OFPFW_DL_DST
	}
	if w.EtherType {
		v = v | OFPFW_DL_TYPE
	}
	if w.Protocol {
		v = v | OFPFW_NW_PROTO
	}
	if w.SrcPort {
		v = v | OFPFW_TP_SRC
	}
	if w.DstPort {
		v = v | OFPFW_TP_DST
	}
	if w.SrcIP > 0 {
		v = v | (uint32(w.SrcIP) << 8)
	}
	if w.DstIP > 0 {
		v = v | (uint32(w.DstIP) << 14)
	}
	if w.VLANPriority {
		v = v | OFPFW_DL_VLAN_PCP
	}
	if w.TOS {
		v = v | OFPFW_NW_TOS
	}

	return v, nil
}

func unmarshalFlowWildcard(w uint32) *FlowWildcard {
	v := &FlowWildcard{}

	if w&OFPFW_IN_PORT != 0 {
		v.InPort = true
	}
	if w&OFPFW_DL_VLAN != 0 {
		v.VLANID = true
	}
	if w&OFPFW_DL_SRC != 0 {
		v.SrcMAC = true
	}
	if w&OFPFW_DL_DST != 0 {
		v.DstMAC = true
	}
	if w&OFPFW_DL_TYPE != 0 {
		v.EtherType = true
	}
	if w&OFPFW_NW_PROTO != 0 {
		v.Protocol = true
	}
	if w&OFPFW_TP_SRC != 0 {
		v.SrcPort = true
	}
	if w&OFPFW_TP_DST != 0 {
		v.DstPort = true
	}
	v.SrcIP = uint8((w & (uint32(0x3F) << 8)) >> 8)
	v.DstIP = uint8((w & (uint32(0x3F) << 14)) >> 14)
	if w&OFPFW_DL_VLAN_PCP != 0 {
		v.VLANPriority = true
	}
	if w&OFPFW_NW_TOS != 0 {
		v.TOS = true
	}

	return v
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

// TODO: other setters and getters for FlowMatch

func (r *FlowMatch) MarshalBinary() ([]byte, error) {
	wildcard, err := marshalFlowWildcard(r.wildcards)
	if err != nil {
		return nil, err
	}

	data := make([]byte, 40)
	binary.BigEndian.PutUint32(data[0:4], wildcard)
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
	nwSrc, n := binary.Uvarint(r.srcIP.To4())
	if n <= 0 {
		return nil, errors.New("invalid source IP address")
	}
	binary.BigEndian.PutUint32(data[28:32], uint32(nwSrc))
	nwDst, n := binary.Uvarint(r.dstIP.To4())
	if n <= 0 {
		return nil, errors.New("invalid destination IP address")
	}
	binary.BigEndian.PutUint32(data[32:36], uint32(nwDst))
	binary.BigEndian.PutUint16(data[36:38], r.srcPort)
	binary.BigEndian.PutUint16(data[38:40], r.dstPort)

	return data, nil
}

func (r *FlowMatch) UnmarshalBinary(data []byte) error {
	if len(data) != 40 {
		return ErrInvalidPacketLength
	}

	r.wildcards = unmarshalFlowWildcard(binary.BigEndian.Uint32(data[0:4]))
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
	// TODO: Test that big-endian representation for IP is correct
	nwSrc := binary.BigEndian.Uint32(data[28:32])
	r.srcIP = net.IPv4(byte(nwSrc>>24&0xFF), byte(nwSrc>>16&0xFF), byte(nwSrc>>8&0xFF), byte(nwSrc&0xFF))
	nwDst := binary.BigEndian.Uint32(data[32:36])
	r.dstIP = net.IPv4(byte(nwDst>>24&0xFF), byte(nwDst>>16&0xFF), byte(nwDst>>8&0xFF), byte(nwDst&0xFF))
	r.srcPort = binary.BigEndian.Uint16(data[36:38])
	r.dstPort = binary.BigEndian.Uint16(data[38:40])

	return nil
}
