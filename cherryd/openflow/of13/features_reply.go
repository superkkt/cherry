/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

import (
	"encoding/binary"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type FeaturesReply struct {
	header       openflow.Header
	DPID         uint64
	NumBuffers   uint32
	NumTables    uint8
	AuxID        uint8
	Capabilities Capability
}

type Capability struct {
	OFPC_FLOW_STATS   bool /* Flow statistics. */
	OFPC_TABLE_STATS  bool /* Table statistics. */
	OFPC_PORT_STATS   bool /* Port statistics. */
	OFPC_GROUP_STATS  bool /* Group statistics. */
	OFPC_IP_REASM     bool /* Can reassemble IP fragments. */
	OFPC_QUEUE_STATS  bool /* Queue statistics. */
	OFPC_PORT_BLOCKED bool /* Switch will block looping ports. */
}

func (r *FeaturesReply) Header() openflow.Header {
	return r.header
}

func (r *FeaturesReply) MarshalBinary() ([]byte, error) {
	return nil, openflow.ErrInvalidMarshaling
}

func getCapability(capabilities uint32) Capability {
	return Capability{
		OFPC_FLOW_STATS:   capabilities&OFPC_FLOW_STATS != 0,
		OFPC_TABLE_STATS:  capabilities&OFPC_TABLE_STATS != 0,
		OFPC_PORT_STATS:   capabilities&OFPC_PORT_STATS != 0,
		OFPC_GROUP_STATS:  capabilities&OFPC_GROUP_STATS != 0,
		OFPC_IP_REASM:     capabilities&OFPC_IP_REASM != 0,
		OFPC_QUEUE_STATS:  capabilities&OFPC_QUEUE_STATS != 0,
		OFPC_PORT_BLOCKED: capabilities&OFPC_PORT_BLOCKED != 0,
	}
}

func (r *FeaturesReply) UnmarshalBinary(data []byte) error {
	if err := r.header.UnmarshalBinary(data); err != nil {
		return err
	}
	if len(data) < int(r.header.Length) {
		return openflow.ErrInvalidPacketLength
	}

	r.DPID = binary.BigEndian.Uint64(data[8:16])
	r.NumBuffers = binary.BigEndian.Uint32(data[16:20])
	r.NumTables = data[20]
	r.AuxID = data[21]
	r.Capabilities = getCapability(binary.BigEndian.Uint32(data[24:28]))

	return nil
}
