/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding/binary"
)

type FeaturesReplyMessage struct {
	Header
	DPID         uint64
	NumBuffers   uint32
	NumTables    uint8
	capabilities uint32
	actions      uint32
	Ports        []Port
}

type Capability struct {
	OFPC_FLOW_STATS   bool /* Flow statistics. */
	OFPC_TABLE_STATS  bool /* Table statistics. */
	OFPC_PORT_STATS   bool /* Port statistics. */
	OFPC_STP          bool /* 802.1d spanning tree. */
	OFPC_RESERVED     bool /* Reserved, must be zero. */
	OFPC_IP_REASM     bool /* Can reassemble IP fragments. */
	OFPC_QUEUE_STATS  bool /* Queue statistics. */
	OFPC_ARP_MATCH_IP bool /* Match IP addresses in ARP pkts. */
}

func (r *FeaturesReplyMessage) GetCapability() *Capability {
	return &Capability{
		OFPC_FLOW_STATS:   r.capabilities&OFPC_FLOW_STATS != 0,
		OFPC_TABLE_STATS:  r.capabilities&OFPC_TABLE_STATS != 0,
		OFPC_PORT_STATS:   r.capabilities&OFPC_PORT_STATS != 0,
		OFPC_STP:          r.capabilities&OFPC_STP != 0,
		OFPC_RESERVED:     r.capabilities&OFPC_RESERVED != 0,
		OFPC_IP_REASM:     r.capabilities&OFPC_IP_REASM != 0,
		OFPC_QUEUE_STATS:  r.capabilities&OFPC_QUEUE_STATS != 0,
		OFPC_ARP_MATCH_IP: r.capabilities&OFPC_ARP_MATCH_IP != 0,
	}
}

type Action struct {
	OFPAT_OUTPUT       bool /* Output to switch port. */
	OFPAT_SET_VLAN_VID bool /* Set the 802.1q VLAN id. */
	OFPAT_SET_VLAN_PCP bool /* Set the 802.1q priority. */
	OFPAT_STRIP_VLAN   bool /* Strip the 802.1q header. */
	OFPAT_SET_DL_SRC   bool /* Ethernet source address. */
	OFPAT_SET_DL_DST   bool /* Ethernet destination address. */
	OFPAT_SET_NW_SRC   bool /* IP source address. */
	OFPAT_SET_NW_DST   bool /* IP destination address. */
	OFPAT_SET_NW_TOS   bool /* IP ToS (DSCP field, 6 bits). */
	OFPAT_SET_TP_SRC   bool /* TCP/UDP source port. */
	OFPAT_SET_TP_DST   bool /* TCP/UDP destination port. */
	OFPAT_ENQUEUE      bool /* Output to queue. */
}

func (r *FeaturesReplyMessage) GetSupportedAction() *Action {
	return &Action{
		OFPAT_OUTPUT:       r.actions&(1<<OFPAT_OUTPUT) != 0,
		OFPAT_SET_VLAN_VID: r.actions&(1<<OFPAT_SET_VLAN_VID) != 0,
		OFPAT_SET_VLAN_PCP: r.actions&(1<<OFPAT_SET_VLAN_PCP) != 0,
		OFPAT_STRIP_VLAN:   r.actions&(1<<OFPAT_STRIP_VLAN) != 0,
		OFPAT_SET_DL_SRC:   r.actions&(1<<OFPAT_SET_DL_SRC) != 0,
		OFPAT_SET_DL_DST:   r.actions&(1<<OFPAT_SET_DL_DST) != 0,
		OFPAT_SET_NW_SRC:   r.actions&(1<<OFPAT_SET_NW_SRC) != 0,
		OFPAT_SET_NW_DST:   r.actions&(1<<OFPAT_SET_NW_DST) != 0,
		OFPAT_SET_NW_TOS:   r.actions&(1<<OFPAT_SET_NW_TOS) != 0,
		OFPAT_SET_TP_SRC:   r.actions&(1<<OFPAT_SET_TP_SRC) != 0,
		OFPAT_SET_TP_DST:   r.actions&(1<<OFPAT_SET_TP_DST) != 0,
		OFPAT_ENQUEUE:      r.actions&(1<<OFPAT_ENQUEUE) != 0,
	}
}

func (r *FeaturesReplyMessage) UnmarshalBinary(data []byte) error {
	header := &Header{}
	if err := header.UnmarshalBinary(data); err != nil {
		return err
	}
	if len(data) != int(header.Length) {
		return ErrInvalidPacketLength
	}

	r.Header = *header
	r.DPID = binary.BigEndian.Uint64(data[8:16])
	r.NumBuffers = binary.BigEndian.Uint32(data[16:20])
	r.NumTables = data[20]
	r.capabilities = binary.BigEndian.Uint32(data[24:28])
	r.actions = binary.BigEndian.Uint32(data[28:32])

	nPorts := (header.Length - 32) / 48
	if nPorts == 0 {
		return nil
	}
	r.Ports = make([]Port, nPorts)
	for i := uint16(0); i < nPorts; i++ {
		buf := data[32+i*48:]
		p := Port{}
		if err := p.UnmarshalBinary(buf[0:48]); err != nil {
			return err
		}
		r.Ports[i] = p
	}

	return nil
}
