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
}

type FlowActionOutput struct {
	Port   uint16
	maxLen uint16
}

func (r *FlowActionOutput) GetActionType() ActionType {
	return OFPAT_OUTPUT
}

func (r *FlowActionOutput) MarshalBinary() ([]byte, error) {
	v := make([]byte, 16)
	binary.BigEndian.PutUint16(v[0:2], uint16(OFPAT_OUTPUT))
	binary.BigEndian.PutUint16(v[2:4], 16)
	binary.BigEndian.PutUint16(v[8:10], uint16(OFPAT_OUTPUT))
	binary.BigEndian.PutUint16(v[10:12], 8)
	binary.BigEndian.PutUint16(v[12:14], r.Port)
	// We don't support buffer ID and partial PACKET_IN
	r.maxLen = 65535
	binary.BigEndian.PutUint16(v[14:16], r.maxLen)

	return v, nil
}

type FlowActionEnqueue struct {
	Port    uint16
	QueueID uint32
}

func (r *FlowActionEnqueue) GetActionType() ActionType {
	return OFPAT_ENQUEUE
}

func (r *FlowActionEnqueue) MarshalBinary() ([]byte, error) {
	v := make([]byte, 24)
	binary.BigEndian.PutUint16(v[0:2], uint16(OFPAT_ENQUEUE))
	binary.BigEndian.PutUint16(v[2:4], 24)
	binary.BigEndian.PutUint16(v[8:10], uint16(OFPAT_ENQUEUE))
	binary.BigEndian.PutUint16(v[10:12], 16)
	binary.BigEndian.PutUint16(v[12:14], r.Port)
	binary.BigEndian.PutUint32(v[20:24], r.QueueID)

	return v, nil
}

type FlowActionSetVLANID struct {
	ID uint16
}

func (r *FlowActionSetVLANID) GetActionType() ActionType {
	return OFPAT_SET_VLAN_VID
}

func (r *FlowActionSetVLANID) MarshalBinary() ([]byte, error) {
	v := make([]byte, 16)
	binary.BigEndian.PutUint16(v[0:2], uint16(OFPAT_SET_VLAN_VID))
	binary.BigEndian.PutUint16(v[2:4], 16)
	binary.BigEndian.PutUint16(v[8:10], uint16(OFPAT_SET_VLAN_VID))
	binary.BigEndian.PutUint16(v[10:12], 8)
	binary.BigEndian.PutUint16(v[12:14], r.ID)

	return v, nil
}

type FlowActionSetVLANPriority struct {
	Priority uint8
}

func (r *FlowActionSetVLANPriority) GetActionType() ActionType {
	return OFPAT_SET_VLAN_PCP
}

func (r *FlowActionSetVLANPriority) MarshalBinary() ([]byte, error) {
	v := make([]byte, 16)
	binary.BigEndian.PutUint16(v[0:2], uint16(OFPAT_SET_VLAN_PCP))
	binary.BigEndian.PutUint16(v[2:4], 16)
	binary.BigEndian.PutUint16(v[8:10], uint16(OFPAT_SET_VLAN_PCP))
	binary.BigEndian.PutUint16(v[10:12], 8)
	v[12] = r.Priority

	return v, nil
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

func marshalMAC(t ActionType, mac net.HardwareAddr) []byte {
	v := make([]byte, 24)
	binary.BigEndian.PutUint16(v[0:2], uint16(t))
	binary.BigEndian.PutUint16(v[2:4], 24)
	binary.BigEndian.PutUint16(v[8:10], uint16(t))
	binary.BigEndian.PutUint16(v[10:12], 16)
	copy(v[12:18], mac)

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

func marshalIP(t ActionType, ip net.IP) []byte {
	v := make([]byte, 16)
	binary.BigEndian.PutUint16(v[0:2], uint16(t))
	binary.BigEndian.PutUint16(v[2:4], 16)
	binary.BigEndian.PutUint16(v[8:10], uint16(t))
	binary.BigEndian.PutUint16(v[10:12], 8)
	// TODO: Test that big-endian representation for IP is correct
	ipInt, n := binary.Uvarint(ip.To4())
	if n <= 0 {
		panic("Invalid IP address!")
	}
	binary.BigEndian.PutUint32(v[12:16], uint32(ipInt))

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

type FlowActionSetTOS struct {
	TOS uint8
}

func (r *FlowActionSetTOS) GetActionType() ActionType {
	return OFPAT_SET_NW_TOS
}

func (r *FlowActionSetTOS) MarshalBinary() ([]byte, error) {
	v := make([]byte, 16)
	binary.BigEndian.PutUint16(v[0:2], uint16(OFPAT_SET_NW_TOS))
	binary.BigEndian.PutUint16(v[2:4], 16)
	binary.BigEndian.PutUint16(v[8:10], uint16(OFPAT_SET_NW_TOS))
	binary.BigEndian.PutUint16(v[10:12], 8)
	v[12] = r.TOS

	return v, nil
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

func marshalPort(t ActionType, port uint16) []byte {
	v := make([]byte, 16)
	binary.BigEndian.PutUint16(v[0:2], uint16(t))
	binary.BigEndian.PutUint16(v[2:4], 16)
	binary.BigEndian.PutUint16(v[8:10], uint16(t))
	binary.BigEndian.PutUint16(v[10:12], 8)
	binary.BigEndian.PutUint16(v[12:14], port)

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
