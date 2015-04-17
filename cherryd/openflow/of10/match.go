/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of10

import (
	"encoding/binary"
	"errors"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"net"
)

var (
	ErrInvalidMAC            = errors.New("invalid MAC address")
	ErrInvalidIP             = errors.New("invalid IP address")
	ErrUnsupportedIPProtocol = errors.New("Unsupported IP protocol")
	ErrUnsupportedEtherType  = errors.New("Unsupported Ethernet type")
)

type Wildcard struct {
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
}

func newWildcardAll() *Wildcard {
	return &Wildcard{
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
	}
}

func (r *Wildcard) MarshalBinary() ([]byte, error) {
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

	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data[0:4], v)

	return data, nil
}

func (r *Wildcard) UnmarshalBinary(data []byte) error {
	if data == nil || len(data) < 4 {
		return openflow.ErrInvalidPacketLength
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

	return nil
}

type Match struct {
	wildcards    *Wildcard
	inPort       uint16
	srcMAC       net.HardwareAddr
	dstMAC       net.HardwareAddr
	vlanID       uint16
	vlanPriority uint8
	etherType    uint16
	protocol     uint8
	srcIP        net.IP
	dstIP        net.IP
	srcPort      uint16
	dstPort      uint16
}

// NewMatch returns a Match whose fields are all wildcarded
func NewMatch() *Match {
	return &Match{
		wildcards: newWildcardAll(),
		srcMAC:    openflow.ZeroMAC,
		dstMAC:    openflow.ZeroMAC,
		srcIP:     openflow.ZeroIP,
		dstIP:     openflow.ZeroIP,
	}
}

func (r *Match) SetWildcardSrcPort() error {
	r.srcPort = 0
	r.wildcards.SrcPort = true
	return nil
}

func (r *Match) SetSrcPort(p uint16) error {
	// IPv4?
	if r.etherType != 0x0800 {
		return ErrUnsupportedEtherType
	}
	// TCP or UDP?
	if r.protocol != 0x06 && r.protocol != 0x11 {
		return ErrUnsupportedIPProtocol
	}

	r.srcPort = p
	r.wildcards.SrcPort = false
	return nil
}

func (r *Match) SrcPort() (wildcard bool, port uint16) {
	return r.wildcards.SrcPort, r.srcPort
}

func (r *Match) SetWildcardDstPort() error {
	r.dstPort = 0
	r.wildcards.DstPort = true
	return nil
}

func (r *Match) SetDstPort(p uint16) error {
	// IPv4?
	if r.etherType != 0x0800 {
		return ErrUnsupportedEtherType
	}
	// TCP or UDP?
	if r.protocol != 0x06 && r.protocol != 0x11 {
		return ErrUnsupportedIPProtocol
	}

	r.dstPort = p
	r.wildcards.DstPort = false
	return nil
}

func (r *Match) DstPort() (wildcard bool, port uint16) {
	return r.wildcards.DstPort, r.dstPort
}

func (r *Match) SetWildcardVLANID() error {
	r.vlanID = 0
	r.wildcards.VLANID = true
	return nil
}

func (r *Match) SetVLANID(id uint16) error {
	r.vlanID = id
	r.wildcards.VLANID = false
	return nil
}

func (r *Match) VLANID() (wildcard bool, vlanID uint16) {
	return r.wildcards.VLANID, r.vlanID
}

func (r *Match) SetWildcardVLANPriority() error {
	r.vlanPriority = 0
	r.wildcards.VLANPriority = true
	return nil
}

func (r *Match) SetVLANPriority(p uint8) error {
	r.vlanPriority = p
	r.wildcards.VLANPriority = false
	return nil
}

func (r *Match) VLANPriority() (wildcard bool, priority uint8) {
	return r.wildcards.VLANPriority, r.vlanPriority
}

func (r *Match) SetWildcardIPProtocol() error {
	r.protocol = 0
	r.wildcards.Protocol = true
	return nil
}

func (r *Match) SetIPProtocol(p uint8) error {
	// IPv4?
	if r.etherType != 0x0800 {
		return ErrUnsupportedEtherType
	}

	r.protocol = p
	r.wildcards.Protocol = false
	return nil
}

func (r *Match) IPProtocol() (wildcard bool, protocol uint8) {
	return r.wildcards.Protocol, r.protocol
}

func (r *Match) SetWildcardInPort() error {
	r.inPort = 0
	r.wildcards.InPort = true
	return nil
}

func (r *Match) SetInPort(port uint32) error {
	r.inPort = uint16(port)
	r.wildcards.InPort = false
	return nil
}

func (r *Match) InPort() (wildcard bool, inport uint32) {
	return r.wildcards.InPort, uint32(r.inPort)
}

func (r *Match) SetWildcardSrcMAC() error {
	r.srcMAC = openflow.ZeroMAC
	r.wildcards.SrcMAC = true
	return nil
}

func (r *Match) SetSrcMAC(mac net.HardwareAddr) error {
	if mac == nil || len(mac) == 0 {
		return ErrInvalidMAC
	}
	r.srcMAC = make([]byte, len(mac))
	copy(r.srcMAC, mac)
	r.wildcards.SrcMAC = false

	return nil
}

func (r *Match) SrcMAC() (wildcard bool, mac net.HardwareAddr) {
	return r.wildcards.SrcMAC, r.srcMAC
}

func (r *Match) SetWildcardDstMAC() error {
	r.dstMAC = openflow.ZeroMAC
	r.wildcards.DstMAC = true
	return nil
}

func (r *Match) SetDstMAC(mac net.HardwareAddr) error {
	if mac == nil || len(mac) == 0 {
		return ErrInvalidMAC
	}
	r.dstMAC = make([]byte, len(mac))
	copy(r.dstMAC, mac)
	r.wildcards.DstMAC = false

	return nil
}

func (r *Match) DstMAC() (wildcard bool, mac net.HardwareAddr) {
	return r.wildcards.DstMAC, r.dstMAC
}

func (r *Match) SetSrcIP(ip *net.IPNet) error {
	if ip == nil || ip.IP == nil || len(ip.IP) == 0 {
		return ErrInvalidIP
	}
	// IPv4?
	if r.etherType != 0x0800 {
		return ErrUnsupportedEtherType
	}

	r.srcIP = make([]byte, len(ip.IP))
	copy(r.srcIP, ip.IP)

	netmaskBits, _ := ip.Mask.Size()
	if netmaskBits >= 32 {
		r.wildcards.SrcIP = 0
	} else {
		r.wildcards.SrcIP = uint8(32 - netmaskBits)
	}

	return nil
}

func (r *Match) SrcIP() *net.IPNet {
	ip := make([]byte, len(r.srcIP))
	copy(ip, r.srcIP)

	return &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(32-int(r.wildcards.SrcIP), 32),
	}
}

func (r *Match) SetDstIP(ip *net.IPNet) error {
	if ip == nil || ip.IP == nil || len(ip.IP) == 0 {
		return ErrInvalidIP
	}
	// IPv4?
	if r.etherType != 0x0800 {
		return ErrUnsupportedEtherType
	}

	r.dstIP = make([]byte, len(ip.IP))
	copy(r.dstIP, ip.IP)

	netmaskBits, _ := ip.Mask.Size()
	if netmaskBits >= 32 {
		r.wildcards.DstIP = 0
	} else {
		r.wildcards.DstIP = uint8(32 - netmaskBits)
	}

	return nil
}

func (r *Match) DstIP() *net.IPNet {
	ip := make([]byte, len(r.dstIP))
	copy(ip, r.dstIP)

	return &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(32-int(r.wildcards.DstIP), 32),
	}
}

func (r *Match) SetWildcardEtherType() error {
	r.etherType = 0
	r.wildcards.EtherType = true
	return nil
}

func (r *Match) SetEtherType(t uint16) error {
	r.etherType = t
	r.wildcards.EtherType = false
	return nil
}

func (r *Match) EtherType() (wildcard bool, etherType uint16) {
	return r.wildcards.EtherType, r.etherType
}

func (r *Match) MarshalBinary() ([]byte, error) {
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
	data[25] = r.protocol
	// data[26:28] = padding
	copy(data[28:32], []byte(r.srcIP.To4()))
	copy(data[32:36], []byte(r.dstIP.To4()))
	binary.BigEndian.PutUint16(data[36:38], r.srcPort)
	binary.BigEndian.PutUint16(data[38:40], r.dstPort)

	return data, nil
}

func (r *Match) UnmarshalBinary(data []byte) error {
	if len(data) < 40 {
		return openflow.ErrInvalidPacketLength
	}

	r.wildcards = new(Wildcard)
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
	// data[21] = padding
	r.etherType = binary.BigEndian.Uint16(data[22:24])
	r.protocol = data[25]
	// data[26:28] = padding
	r.srcIP = net.IPv4(data[28], data[29], data[30], data[31])
	r.dstIP = net.IPv4(data[32], data[33], data[34], data[35])
	r.srcPort = binary.BigEndian.Uint16(data[36:38])
	r.dstPort = binary.BigEndian.Uint16(data[38:40])

	return nil
}
