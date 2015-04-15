/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of10

import (
	"encoding/binary"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type FeaturesReply struct {
	openflow.Message
	DPID         uint64
	NumBuffers   uint32
	NumTables    uint8
	Capabilities Capability
	Actions      SupportedAction
	Ports        []*Port
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

type SupportedAction struct {
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

func getSupportedAction(actions uint32) SupportedAction {
	return SupportedAction{
		OFPAT_OUTPUT:       actions&(1<<OFPAT_OUTPUT) != 0,
		OFPAT_SET_VLAN_VID: actions&(1<<OFPAT_SET_VLAN_VID) != 0,
		OFPAT_SET_VLAN_PCP: actions&(1<<OFPAT_SET_VLAN_PCP) != 0,
		OFPAT_STRIP_VLAN:   actions&(1<<OFPAT_STRIP_VLAN) != 0,
		OFPAT_SET_DL_SRC:   actions&(1<<OFPAT_SET_DL_SRC) != 0,
		OFPAT_SET_DL_DST:   actions&(1<<OFPAT_SET_DL_DST) != 0,
		OFPAT_SET_NW_SRC:   actions&(1<<OFPAT_SET_NW_SRC) != 0,
		OFPAT_SET_NW_DST:   actions&(1<<OFPAT_SET_NW_DST) != 0,
		OFPAT_SET_NW_TOS:   actions&(1<<OFPAT_SET_NW_TOS) != 0,
		OFPAT_SET_TP_SRC:   actions&(1<<OFPAT_SET_TP_SRC) != 0,
		OFPAT_SET_TP_DST:   actions&(1<<OFPAT_SET_TP_DST) != 0,
		OFPAT_ENQUEUE:      actions&(1<<OFPAT_ENQUEUE) != 0,
	}
}

func getCapability(capabilities uint32) Capability {
	return Capability{
		OFPC_FLOW_STATS:   capabilities&OFPC_FLOW_STATS != 0,
		OFPC_TABLE_STATS:  capabilities&OFPC_TABLE_STATS != 0,
		OFPC_PORT_STATS:   capabilities&OFPC_PORT_STATS != 0,
		OFPC_STP:          capabilities&OFPC_STP != 0,
		OFPC_RESERVED:     capabilities&OFPC_RESERVED != 0,
		OFPC_IP_REASM:     capabilities&OFPC_IP_REASM != 0,
		OFPC_QUEUE_STATS:  capabilities&OFPC_QUEUE_STATS != 0,
		OFPC_ARP_MATCH_IP: capabilities&OFPC_ARP_MATCH_IP != 0,
	}
}

func (r *FeaturesReply) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 24 {
		return openflow.ErrInvalidPacketLength
	}
	r.DPID = binary.BigEndian.Uint64(payload[0:8])
	r.NumBuffers = binary.BigEndian.Uint32(payload[8:12])
	r.NumTables = payload[12]
	r.Capabilities = getCapability(binary.BigEndian.Uint32(payload[16:20]))
	r.Actions = getSupportedAction(binary.BigEndian.Uint32(payload[20:24]))

	nPorts := (len(payload) - 24) / 48
	if nPorts == 0 {
		return nil
	}
	r.Ports = make([]*Port, nPorts)
	for i := 0; i < nPorts; i++ {
		buf := payload[24+i*48:]
		r.Ports[i] = new(Port)
		if err := r.Ports[i].UnmarshalBinary(buf[0:48]); err != nil {
			return err
		}
	}

	return nil
}
