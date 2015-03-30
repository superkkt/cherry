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

type FlowAction interface {
	GetActionType() ActionType
	MarshalBinary() ([]byte, error)
	UnmarshalBinary(data []byte) error
}

type FlowActionOutput struct {
	Port   PortNumber
	MaxLen uint16
}

func (r *FlowActionOutput) GetActionType() ActionType {
	return OFPAT_OUTPUT
}

func (r *FlowActionOutput) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	binary.BigEndian.PutUint16(v[0:2], uint16(OFPAT_OUTPUT))
	binary.BigEndian.PutUint16(v[2:4], 8)
	binary.BigEndian.PutUint16(v[4:6], uint16(r.Port))
	// We don't support buffer ID and partial PACKET_IN
	binary.BigEndian.PutUint16(v[6:8], 65535)

	return v, nil
}

func (r *FlowActionOutput) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return ErrInvalidPacketLength
	}

	r.Port = PortNumber(binary.BigEndian.Uint16(data[4:6]))
	r.MaxLen = binary.BigEndian.Uint16(data[6:8])

	return nil
}

type FlowActionEnqueue struct {
	Port    uint16
	QueueID uint32
}

func (r *FlowActionEnqueue) GetActionType() ActionType {
	return OFPAT_ENQUEUE
}

func (r *FlowActionEnqueue) MarshalBinary() ([]byte, error) {
	v := make([]byte, 16)
	binary.BigEndian.PutUint16(v[0:2], uint16(OFPAT_ENQUEUE))
	binary.BigEndian.PutUint16(v[2:4], 16)
	binary.BigEndian.PutUint16(v[4:6], r.Port)
	binary.BigEndian.PutUint32(v[12:16], r.QueueID)

	return v, nil
}

func (r *FlowActionEnqueue) UnmarshalBinary(data []byte) error {
	if len(data) < 16 {
		return ErrInvalidPacketLength
	}

	r.Port = binary.BigEndian.Uint16(data[4:6])
	r.QueueID = binary.BigEndian.Uint32(data[12:16])

	return nil
}

type FlowActionSetVLANID struct {
	ID uint16
}

func (r *FlowActionSetVLANID) GetActionType() ActionType {
	return OFPAT_SET_VLAN_VID
}

func (r *FlowActionSetVLANID) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	binary.BigEndian.PutUint16(v[0:2], uint16(OFPAT_SET_VLAN_VID))
	binary.BigEndian.PutUint16(v[2:4], 8)
	binary.BigEndian.PutUint16(v[4:6], r.ID)

	return v, nil
}

func (r *FlowActionSetVLANID) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return ErrInvalidPacketLength
	}

	r.ID = binary.BigEndian.Uint16(data[4:6])

	return nil
}

type FlowActionSetVLANPriority struct {
	Priority uint8
}

func (r *FlowActionSetVLANPriority) GetActionType() ActionType {
	return OFPAT_SET_VLAN_PCP
}

func (r *FlowActionSetVLANPriority) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	binary.BigEndian.PutUint16(v[0:2], uint16(OFPAT_SET_VLAN_PCP))
	binary.BigEndian.PutUint16(v[2:4], 8)
	v[4] = r.Priority

	return v, nil
}

func (r *FlowActionSetVLANPriority) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return ErrInvalidPacketLength
	}

	r.Priority = data[4]

	return nil
}

type FlowActionStripVLAN struct{}

func (r *FlowActionStripVLAN) GetActionType() ActionType {
	return OFPAT_STRIP_VLAN
}

func (r *FlowActionStripVLAN) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	binary.BigEndian.PutUint16(v[0:2], uint16(OFPAT_STRIP_VLAN))
	binary.BigEndian.PutUint16(v[2:4], 8)

	return v, nil
}

func (r *FlowActionStripVLAN) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return ErrInvalidPacketLength
	}

	return nil
}

type FlowActionSetSrcMAC struct {
	MAC net.HardwareAddr
}

func (r *FlowActionSetSrcMAC) GetActionType() ActionType {
	return OFPAT_SET_DL_SRC
}

func (r *FlowActionSetSrcMAC) MarshalBinary() ([]byte, error) {
	if len(r.MAC) != 6 {
		return nil, errors.New("invalid MAC address of FlowActionSetSrcMAC")
	}

	return marshalMAC(OFPAT_SET_DL_SRC, r.MAC), nil
}

func (r *FlowActionSetSrcMAC) UnmarshalBinary(data []byte) error {
	if len(data) < 16 {
		return ErrInvalidPacketLength
	}

	// FIXME: Is this okay?
	r.MAC = make([]byte, 6)
	copy(r.MAC, data[4:10])

	return nil
}

func marshalMAC(t ActionType, mac net.HardwareAddr) []byte {
	v := make([]byte, 16)
	binary.BigEndian.PutUint16(v[0:2], uint16(t))
	binary.BigEndian.PutUint16(v[2:4], 16)
	copy(v[4:10], mac)

	return v
}

type FlowActionSetDstMAC struct {
	MAC net.HardwareAddr
}

func (r *FlowActionSetDstMAC) GetActionType() ActionType {
	return OFPAT_SET_DL_DST
}

func (r *FlowActionSetDstMAC) MarshalBinary() ([]byte, error) {
	if len(r.MAC) != 6 {
		return nil, errors.New("invalid MAC address of FlowActionSetDstMAC")
	}

	return marshalMAC(OFPAT_SET_DL_DST, r.MAC), nil
}

func (r *FlowActionSetDstMAC) UnmarshalBinary(data []byte) error {
	if len(data) < 16 {
		return ErrInvalidPacketLength
	}

	// FIXME: Is this okay?
	r.MAC = make([]byte, 6)
	copy(r.MAC, data[4:10])

	return nil
}

type FlowActionSetSrcIP struct {
	IP net.IP
}

func (r *FlowActionSetSrcIP) GetActionType() ActionType {
	return OFPAT_SET_NW_SRC
}

func (r *FlowActionSetSrcIP) MarshalBinary() ([]byte, error) {
	if len(r.IP) != 4 && len(r.IP) != 16 {
		return nil, errors.New("invalid IP address of FlowActionSetSrcIP")
	}

	return marshalIP(OFPAT_SET_NW_SRC, r.IP), nil
}

func (r *FlowActionSetSrcIP) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return ErrInvalidPacketLength
	}

	r.IP = net.IPv4(data[4], data[5], data[6], data[7])

	return nil
}

func marshalIP(t ActionType, ip net.IP) []byte {
	v := make([]byte, 8)
	binary.BigEndian.PutUint16(v[0:2], uint16(t))
	binary.BigEndian.PutUint16(v[2:4], 8)
	// TODO: Test that big-endian representation for IP is correct
	copy(v[4:8], []byte(ip.To4()))

	return v
}

type FlowActionSetDstIP struct {
	IP net.IP
}

func (r *FlowActionSetDstIP) GetActionType() ActionType {
	return OFPAT_SET_NW_DST
}

func (r *FlowActionSetDstIP) MarshalBinary() ([]byte, error) {
	if len(r.IP) != 4 && len(r.IP) != 16 {
		return nil, errors.New("invalid IP address of FlowActionSetDstIP")
	}

	return marshalIP(OFPAT_SET_NW_DST, r.IP), nil
}

func (r *FlowActionSetDstIP) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return ErrInvalidPacketLength
	}

	r.IP = net.IPv4(data[4], data[5], data[6], data[7])

	return nil
}

type FlowActionSetTOS struct {
	TOS uint8
}

func (r *FlowActionSetTOS) GetActionType() ActionType {
	return OFPAT_SET_NW_TOS
}

func (r *FlowActionSetTOS) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	binary.BigEndian.PutUint16(v[0:2], uint16(OFPAT_SET_NW_TOS))
	binary.BigEndian.PutUint16(v[2:4], 8)
	v[4] = r.TOS

	return v, nil
}

func (r *FlowActionSetTOS) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return ErrInvalidPacketLength
	}

	r.TOS = data[4]

	return nil
}

type FlowActionSetSrcPort struct {
	Port uint16
}

func (r *FlowActionSetSrcPort) GetActionType() ActionType {
	return OFPAT_SET_TP_SRC
}

func (r *FlowActionSetSrcPort) MarshalBinary() ([]byte, error) {
	return marshalPort(OFPAT_SET_TP_SRC, r.Port), nil
}

func (r *FlowActionSetSrcPort) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return ErrInvalidPacketLength
	}

	r.Port = binary.BigEndian.Uint16(data[4:6])

	return nil
}

func marshalPort(t ActionType, port uint16) []byte {
	v := make([]byte, 8)
	binary.BigEndian.PutUint16(v[0:2], uint16(t))
	binary.BigEndian.PutUint16(v[2:4], 8)
	binary.BigEndian.PutUint16(v[4:6], port)

	return v
}

type FlowActionSetDstPort struct {
	Port uint16
}

func (r *FlowActionSetDstPort) GetActionType() ActionType {
	return OFPAT_SET_TP_DST
}

func (r *FlowActionSetDstPort) MarshalBinary() ([]byte, error) {
	return marshalPort(OFPAT_SET_TP_DST, r.Port), nil
}

func (r *FlowActionSetDstPort) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return ErrInvalidPacketLength
	}

	r.Port = binary.BigEndian.Uint16(data[4:6])

	return nil
}
