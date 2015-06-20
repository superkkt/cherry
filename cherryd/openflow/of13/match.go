/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
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

package of13

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/superkkt/cherry/cherryd/openflow"
	"net"
	"sync"
)

type Match struct {
	err   error
	mutex sync.Mutex
	m     map[uint]interface{}
}

// NewMatch returns a Match whose fields are all wildcarded
func NewMatch() openflow.Match {
	return &Match{
		m: make(map[uint]interface{}),
	}
}

func (r *Match) Error() error {
	return r.err
}

func (r *Match) SetWildcardSrcPort() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.m, OFPXMT_OFB_TCP_SRC)
	delete(r.m, OFPXMT_OFB_UDP_SRC)
}

func (r *Match) SetSrcPort(p uint16) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	etherType, ok := r.m[OFPXMT_OFB_ETH_TYPE]
	if !ok {
		r.err = fmt.Errorf("SetSrcPort: %v", openflow.ErrMissingEtherType)
		return
	}
	// IPv4?
	if etherType.(uint16) != 0x0800 {
		r.err = fmt.Errorf("SetSrcPort: %v", openflow.ErrUnsupportedEtherType)
		return
	}

	proto, ok := r.m[OFPXMT_OFB_IP_PROTO]
	if !ok {
		r.err = fmt.Errorf("SetSrcPort: %v", openflow.ErrMissingIPProtocol)
		return
	}

	switch proto.(uint8) {
	// TCP
	case 0x06:
		r.m[OFPXMT_OFB_TCP_SRC] = p
		delete(r.m, OFPXMT_OFB_UDP_SRC)
	// UDP
	case 0x11:
		r.m[OFPXMT_OFB_UDP_SRC] = p
		delete(r.m, OFPXMT_OFB_TCP_SRC)
	default:
		r.err = fmt.Errorf("SetSrcPort: %v", openflow.ErrUnsupportedIPProtocol)
		return
	}
}

func (r *Match) SrcPort() (wildcard bool, port uint16) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	v, ok := r.m[OFPXMT_OFB_TCP_SRC]
	if ok {
		return false, v.(uint16)
	}

	v, ok = r.m[OFPXMT_OFB_UDP_SRC]
	if ok {
		return false, v.(uint16)
	}

	return true, 0
}

func (r *Match) SetWildcardDstPort() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.m, OFPXMT_OFB_TCP_DST)
	delete(r.m, OFPXMT_OFB_UDP_DST)
}

func (r *Match) SetDstPort(p uint16) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	etherType, ok := r.m[OFPXMT_OFB_ETH_TYPE]
	if !ok {
		r.err = fmt.Errorf("SetDstPort: %v", openflow.ErrMissingEtherType)
		return
	}
	// IPv4?
	if etherType.(uint16) != 0x0800 {
		r.err = fmt.Errorf("SetDstPort: %v", openflow.ErrUnsupportedEtherType)
		return
	}

	proto, ok := r.m[OFPXMT_OFB_IP_PROTO]
	if !ok {
		r.err = fmt.Errorf("SetDstPort: %v", openflow.ErrMissingIPProtocol)
		return
	}

	switch proto.(uint8) {
	// TCP
	case 0x06:
		r.m[OFPXMT_OFB_TCP_DST] = p
		delete(r.m, OFPXMT_OFB_UDP_DST)
	// UDP
	case 0x11:
		r.m[OFPXMT_OFB_UDP_DST] = p
		delete(r.m, OFPXMT_OFB_TCP_DST)
	default:
		r.err = fmt.Errorf("SetDstPort: %v", openflow.ErrUnsupportedIPProtocol)
		return
	}
}

func (r *Match) DstPort() (wildcard bool, port uint16) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	v, ok := r.m[OFPXMT_OFB_TCP_DST]
	if ok {
		return false, v.(uint16)
	}

	v, ok = r.m[OFPXMT_OFB_UDP_DST]
	if ok {
		return false, v.(uint16)
	}

	return true, 0
}

func (r *Match) SetWildcardVLANID() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.m, OFPXMT_OFB_VLAN_VID)
}

func (r *Match) SetVLANID(id uint16) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.m[OFPXMT_OFB_VLAN_VID] = id
}

func (r *Match) VLANID() (wildcard bool, vlanID uint16) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	v, ok := r.m[OFPXMT_OFB_VLAN_VID]
	if ok {
		return false, v.(uint16)
	}

	return true, 0
}

func (r *Match) SetWildcardVLANPriority() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.m, OFPXMT_OFB_VLAN_PCP)
}

func (r *Match) SetVLANPriority(p uint8) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.m[OFPXMT_OFB_VLAN_PCP] = p
}

func (r *Match) VLANPriority() (wildcard bool, priority uint8) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	v, ok := r.m[OFPXMT_OFB_VLAN_PCP]
	if ok {
		return false, v.(uint8)
	}

	return true, 0
}

func (r *Match) SetWildcardIPProtocol() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.m, OFPXMT_OFB_IP_PROTO)
}

func (r *Match) SetIPProtocol(p uint8) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	etherType, ok := r.m[OFPXMT_OFB_ETH_TYPE]
	if !ok {
		r.err = fmt.Errorf("SetIPProtocol: %v", openflow.ErrMissingEtherType)
		return
	}
	// IPv4?
	if etherType.(uint16) != 0x0800 {
		r.err = fmt.Errorf("SetIPProtocol: %v", openflow.ErrUnsupportedEtherType)
		return
	}

	r.m[OFPXMT_OFB_IP_PROTO] = p
}

func (r *Match) IPProtocol() (wildcard bool, protocol uint8) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	v, ok := r.m[OFPXMT_OFB_IP_PROTO]
	if ok {
		return false, v.(uint8)
	}

	return true, 0
}

func (r *Match) SetWildcardInPort() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.m, OFPXMT_OFB_IN_PORT)
}

func (r *Match) SetInPort(port openflow.InPort) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.m[OFPXMT_OFB_IN_PORT] = uint32(port.Value())
}

func (r *Match) InPort() (wildcard bool, inport openflow.InPort) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	v, ok := r.m[OFPXMT_OFB_IN_PORT]
	if ok {
		inport = openflow.NewInPort()
		inport.SetValue(v.(uint32))
		return false, inport
	}

	return true, openflow.NewInPort()
}

func (r *Match) SetWildcardSrcMAC() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.m, OFPXMT_OFB_ETH_SRC)
}

func (r *Match) SetSrcMAC(mac net.HardwareAddr) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if mac == nil {
		panic("mac is nil")
	}
	if len(mac) < 6 {
		r.err = fmt.Errorf("SetSrcMAC: %v", openflow.ErrInvalidMACAddress)
		return
	}
	r.m[OFPXMT_OFB_ETH_SRC] = mac
}

func (r *Match) SrcMAC() (wildcard bool, mac net.HardwareAddr) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	v, ok := r.m[OFPXMT_OFB_ETH_SRC]
	if ok {
		return false, v.(net.HardwareAddr)
	}

	return true, net.HardwareAddr([]byte{0, 0, 0, 0, 0, 0})
}

func (r *Match) SetWildcardDstMAC() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.m, OFPXMT_OFB_ETH_DST)
}

func (r *Match) SetDstMAC(mac net.HardwareAddr) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if mac == nil {
		panic("mac is nil")
	}
	if len(mac) < 6 {
		r.err = fmt.Errorf("SetDstMAC: %v", openflow.ErrInvalidMACAddress)
		return
	}
	r.m[OFPXMT_OFB_ETH_DST] = mac
}

func (r *Match) DstMAC() (wildcard bool, mac net.HardwareAddr) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	v, ok := r.m[OFPXMT_OFB_ETH_DST]
	if ok {
		return false, v.(net.HardwareAddr)
	}

	return true, net.HardwareAddr([]byte{0, 0, 0, 0, 0, 0})
}

func (r *Match) SetSrcIP(ip *net.IPNet) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if ip == nil {
		panic("ip is nil")
	}
	if ip.IP == nil || len(ip.IP) == 0 {
		r.err = fmt.Errorf("SetSrcIP: %v", openflow.ErrInvalidIPAddress)
		return
	}

	etherType, ok := r.m[OFPXMT_OFB_ETH_TYPE]
	if !ok {
		r.err = fmt.Errorf("SetSrcIP: %v", openflow.ErrMissingEtherType)
		return
	}
	// IPv4?
	if etherType.(uint16) != 0x0800 {
		r.err = fmt.Errorf("SetSrcIP: %v", openflow.ErrUnsupportedEtherType)
		return
	}

	r.m[OFPXMT_OFB_IPV4_SRC] = ip
}

func (r *Match) SrcIP() *net.IPNet {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	v, ok := r.m[OFPXMT_OFB_IPV4_SRC]
	if ok {
		return v.(*net.IPNet)
	}

	return &net.IPNet{
		IP:   net.IPv4zero,
		Mask: net.CIDRMask(0, 32),
	}
}

func (r *Match) SetDstIP(ip *net.IPNet) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if ip == nil {
		panic("ip is nil")
	}
	if ip.IP == nil || len(ip.IP) == 0 {
		r.err = fmt.Errorf("SetDstIP: %v", openflow.ErrInvalidIPAddress)
		return
	}

	etherType, ok := r.m[OFPXMT_OFB_ETH_TYPE]
	if !ok {
		r.err = fmt.Errorf("SetDstIP: %v", openflow.ErrMissingEtherType)
		return
	}
	// IPv4?
	if etherType.(uint16) != 0x0800 {
		r.err = fmt.Errorf("SetDstIP: %v", openflow.ErrUnsupportedEtherType)
		return
	}

	r.m[OFPXMT_OFB_IPV4_DST] = ip
}

func (r *Match) DstIP() *net.IPNet {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	v, ok := r.m[OFPXMT_OFB_IPV4_DST]
	if ok {
		return v.(*net.IPNet)
	}

	return &net.IPNet{
		IP:   net.IPv4zero,
		Mask: net.CIDRMask(0, 32),
	}
}

func (r *Match) SetWildcardEtherType() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.m, OFPXMT_OFB_ETH_TYPE)
}

func (r *Match) SetEtherType(t uint16) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.m[OFPXMT_OFB_ETH_TYPE] = t
}

func (r *Match) EtherType() (wildcard bool, etherType uint16) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	v, ok := r.m[OFPXMT_OFB_ETH_TYPE]
	if ok {
		return false, v.(uint16)
	}

	return true, 0
}

func marshalIPNetTLV(field uint8, ip *net.IPNet) ([]byte, error) {
	data := make([]byte, 12)
	// TLV header
	var header uint32 = 0x8000<<16 | uint32(field)<<9 | 0x1<<8 | 8
	binary.BigEndian.PutUint32(data[0:4], header)
	ipv4 := ip.IP.To4()
	if ipv4 == nil {
		return nil, openflow.ErrInvalidIPAddress
	}
	copy(data[4:8], ipv4)
	copy(data[8:12], ip.Mask)
	return data, nil
}

func marshalHardwareAddrTLV(field uint8, mac net.HardwareAddr) ([]byte, error) {
	data := make([]byte, 10)
	// TLV header
	var header uint32 = 0x8000<<16 | uint32(field)<<9 | 0x0<<8 | 6
	binary.BigEndian.PutUint32(data[0:4], header)
	copy(data[4:], mac)
	return data, nil
}

func marshalUint8TLV(field uint8, v uint8) ([]byte, error) {
	data := make([]byte, 5)
	// TLV header
	var header uint32 = 0x8000<<16 | uint32(field)<<9 | 0x0<<8 | 1
	binary.BigEndian.PutUint32(data[0:4], header)
	data[5] = v
	return data, nil
}

func marshalUint16TLV(field uint8, v uint16) ([]byte, error) {
	data := make([]byte, 6)
	// TLV header
	var header uint32 = 0x8000<<16 | uint32(field)<<9 | 0x0<<8 | 2
	binary.BigEndian.PutUint32(data[0:4], header)
	binary.BigEndian.PutUint16(data[4:6], v)
	return data, nil
}

func marshalUint32TLV(field uint8, v uint32) ([]byte, error) {
	data := make([]byte, 8)
	// TLV header
	var header uint32 = 0x8000<<16 | uint32(field)<<9 | 0x0<<8 | 4
	binary.BigEndian.PutUint32(data[0:4], header)
	binary.BigEndian.PutUint32(data[4:8], v)
	return data, nil
}

func marshalTLV(id uint, v interface{}) ([]byte, error) {
	switch id {
	case OFPXMT_OFB_IN_PORT:
		port := v.(uint32)
		return marshalUint32TLV(OFPXMT_OFB_IN_PORT, port)
	case OFPXMT_OFB_ETH_DST:
		mac := v.(net.HardwareAddr)
		return marshalHardwareAddrTLV(OFPXMT_OFB_ETH_DST, mac)
	case OFPXMT_OFB_ETH_SRC:
		mac := v.(net.HardwareAddr)
		return marshalHardwareAddrTLV(OFPXMT_OFB_ETH_SRC, mac)
	case OFPXMT_OFB_ETH_TYPE:
		etherType := v.(uint16)
		return marshalUint16TLV(OFPXMT_OFB_ETH_TYPE, etherType)
	case OFPXMT_OFB_VLAN_VID:
		vid := v.(uint16)
		return marshalUint16TLV(OFPXMT_OFB_VLAN_VID, vid)
	case OFPXMT_OFB_VLAN_PCP:
		priority := v.(uint8)
		return marshalUint8TLV(OFPXMT_OFB_VLAN_PCP, priority)
	case OFPXMT_OFB_IP_PROTO:
		protocol := v.(uint8)
		return marshalUint8TLV(OFPXMT_OFB_IP_PROTO, protocol)
	case OFPXMT_OFB_IPV4_SRC:
		ip := v.(*net.IPNet)
		return marshalIPNetTLV(OFPXMT_OFB_IPV4_SRC, ip)
	case OFPXMT_OFB_IPV4_DST:
		ip := v.(*net.IPNet)
		return marshalIPNetTLV(OFPXMT_OFB_IPV4_DST, ip)
	case OFPXMT_OFB_TCP_SRC:
		port := v.(uint16)
		return marshalUint16TLV(OFPXMT_OFB_TCP_SRC, port)
	case OFPXMT_OFB_TCP_DST:
		port := v.(uint16)
		return marshalUint16TLV(OFPXMT_OFB_TCP_DST, port)
	case OFPXMT_OFB_UDP_SRC:
		port := v.(uint16)
		return marshalUint16TLV(OFPXMT_OFB_UDP_SRC, port)
	case OFPXMT_OFB_UDP_DST:
		port := v.(uint16)
		return marshalUint16TLV(OFPXMT_OFB_UDP_DST, port)
	default:
		panic(fmt.Sprintf("unexpected TLV type: %v", id))
	}
}

func (r *Match) MarshalBinary() ([]byte, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.err != nil {
		return nil, r.err
	}

	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data[0:2], OFPMT_OXM)
	for k, v := range r.m {
		tlv, err := marshalTLV(k, v)
		if err != nil {
			return nil, err
		}
		data = append(data, tlv...)
	}
	// ofp_match.length does not include padding
	binary.BigEndian.PutUint16(data[2:4], uint16(len(data)))
	// Add padding to align as a multiple of 8
	rem := len(data) % 8
	if rem > 0 {
		data = append(data, bytes.Repeat([]byte{0}, 8-rem)...)
	}

	return data, nil
}

func (r *Match) unmarshalUint8TLV(field uint8, data []byte) error {
	if len(data) < 5 {
		return openflow.ErrInvalidPacketLength
	}
	r.m[uint(field)] = data[4]

	return nil
}

func (r *Match) unmarshalUint16TLV(field uint8, data []byte) error {
	if len(data) < 6 {
		return openflow.ErrInvalidPacketLength
	}
	v := binary.BigEndian.Uint16(data[4:6])
	r.m[uint(field)] = v

	return nil
}

func (r *Match) unmarshalUint32TLV(field uint8, data []byte) error {
	if len(data) < 8 {
		return openflow.ErrInvalidPacketLength
	}
	v := binary.BigEndian.Uint32(data[4:8])
	r.m[uint(field)] = v

	return nil
}

func (r *Match) unmarshalHardwareAddrTLV(field uint8, data []byte) error {
	if len(data) < 10 {
		return openflow.ErrInvalidPacketLength
	}
	var mac net.HardwareAddr = make([]byte, 6)
	copy(mac, data[4:10])
	r.m[uint(field)] = mac

	return nil
}

func (r *Match) unmarshalIPNetTLV(field uint8, hasmask uint8, data []byte) error {
	length := 8
	if hasmask == 1 {
		length = 12
	}
	if len(data) < length {
		return openflow.ErrInvalidPacketLength
	}

	ip := net.IPv4(data[4], data[5], data[6], data[7])
	mask := []byte{0, 0, 0, 0}
	if hasmask == 1 {
		mask = []byte{data[8], data[9], data[10], data[11]}
	}

	ipnet := &net.IPNet{
		IP:   ip,
		Mask: mask,
	}
	r.m[uint(field)] = ipnet

	return nil
}

func (r *Match) unmarshalTLV(data []byte) error {
	buf := data
	// TLV header length is 4 bytes
	for len(buf) >= 4 {
		header := binary.BigEndian.Uint32(buf[0:4])
		class := header >> 16 & 0xFFFF
		if class != 0x8000 {
			return errors.New("unsupported TLV class")
		}
		field := header >> 9 & 0x7F
		hasmask := header >> 8 & 0x1
		length := header & 0xFF

		if len(buf) < int(4+length) {
			return openflow.ErrInvalidPacketLength
		}

		switch field {
		case OFPXMT_OFB_IN_PORT:
			if err := r.unmarshalUint32TLV(OFPXMT_OFB_IN_PORT, buf); err != nil {
				return err
			}
		case OFPXMT_OFB_ETH_DST:
			if err := r.unmarshalHardwareAddrTLV(OFPXMT_OFB_ETH_DST, buf); err != nil {
				return err
			}
		case OFPXMT_OFB_ETH_SRC:
			if err := r.unmarshalHardwareAddrTLV(OFPXMT_OFB_ETH_SRC, buf); err != nil {
				return err
			}
		case OFPXMT_OFB_ETH_TYPE:
			if err := r.unmarshalUint16TLV(OFPXMT_OFB_ETH_TYPE, buf); err != nil {
				return err
			}
		case OFPXMT_OFB_VLAN_VID:
			if err := r.unmarshalUint16TLV(OFPXMT_OFB_VLAN_VID, buf); err != nil {
				return err
			}
		case OFPXMT_OFB_VLAN_PCP:
			if err := r.unmarshalUint8TLV(OFPXMT_OFB_VLAN_PCP, buf); err != nil {
				return err
			}
		case OFPXMT_OFB_IP_PROTO:
			if err := r.unmarshalUint8TLV(OFPXMT_OFB_IP_PROTO, buf); err != nil {
				return err
			}
		case OFPXMT_OFB_IPV4_SRC:
			if err := r.unmarshalIPNetTLV(OFPXMT_OFB_IPV4_SRC, uint8(hasmask), buf); err != nil {
				return err
			}
		case OFPXMT_OFB_IPV4_DST:
			if err := r.unmarshalIPNetTLV(OFPXMT_OFB_IPV4_DST, uint8(hasmask), buf); err != nil {
				return err
			}
		case OFPXMT_OFB_TCP_SRC:
			if err := r.unmarshalUint16TLV(OFPXMT_OFB_TCP_SRC, buf); err != nil {
				return err
			}
		case OFPXMT_OFB_TCP_DST:
			if err := r.unmarshalUint16TLV(OFPXMT_OFB_TCP_DST, buf); err != nil {
				return err
			}
		case OFPXMT_OFB_UDP_SRC:
			if err := r.unmarshalUint16TLV(OFPXMT_OFB_UDP_SRC, buf); err != nil {
				return err
			}
		case OFPXMT_OFB_UDP_DST:
			if err := r.unmarshalUint16TLV(OFPXMT_OFB_UDP_DST, buf); err != nil {
				return err
			}
		default:
			// Do nothing
		}

		buf = buf[4+length:]
	}

	return nil
}

func (r *Match) UnmarshalBinary(data []byte) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(data) < 4 {
		return openflow.ErrInvalidPacketLength
	}
	if binary.BigEndian.Uint16(data[0:2]) != OFPMT_OXM {
		return openflow.ErrUnsupportedMatchType
	}
	length := binary.BigEndian.Uint16(data[2:4])
	if len(data) < int(length) {
		return openflow.ErrInvalidPacketLength
	}

	return r.unmarshalTLV(data[4:length])
}
