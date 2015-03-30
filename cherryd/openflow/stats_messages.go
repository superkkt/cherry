/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding/binary"
	"fmt"
	"strings"
)

type StatsRequestMessage struct {
	Header
	Type StatsType
	Body []byte
}

func (r *StatsRequestMessage) MarshalBinary() ([]byte, error) {
	var length uint16 = 12 // header length + type + flags
	if r.Body != nil {
		length += uint16(len(r.Body))
	}

	r.Header.Length = length
	header, err := r.Header.MarshalBinary()
	if err != nil {
		return nil, err
	}

	data := make([]byte, length)
	copy(data[0:8], header)
	binary.BigEndian.PutUint16(data[8:10], uint16(r.Type))
	// flags is not used
	if r.Body != nil && len(r.Body) > 0 {
		copy(data[12:], r.Body)
	}

	return data, nil
}

type StatsReplyMessage struct {
	Header
	Type  StatsType
	Flags uint16
}

func (r *StatsReplyMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 12 {
		return ErrInvalidPacketLength
	}

	header := &Header{}
	if err := header.UnmarshalBinary(data); err != nil {
		return err
	}

	r.Header = *header
	r.Type = StatsType(binary.BigEndian.Uint16(data[8:10]))
	r.Flags = binary.BigEndian.Uint16(data[10:12])

	return nil
}

type DescStatsReplyMessage struct {
	StatsReplyMessage
	Manufacturer string
	Hardware     string
	Software     string
	SerialNumber string
	Description  string
}

func (r *DescStatsReplyMessage) UnmarshalBinary(data []byte) error {
	if len(data) != 1068 {
		return ErrInvalidPacketLength
	}

	reply := StatsReplyMessage{}
	if err := reply.UnmarshalBinary(data); err != nil {
		return err
	}

	r.StatsReplyMessage = reply
	r.Manufacturer = strings.TrimRight(string(data[12:268]), "\x00")
	r.Hardware = strings.TrimRight(string(data[268:524]), "\x00")
	r.Software = strings.TrimRight(string(data[524:780]), "\x00")
	r.SerialNumber = strings.TrimRight(string(data[780:812]), "\x00")
	r.Description = strings.TrimRight(string(data[812:1068]), "\x00")

	return nil
}

type FlowStats struct {
	length          uint16
	TableID         uint8
	Match           *FlowMatch
	DurationSec     uint32
	DurationNanoSec uint32
	Priority        uint16
	IdleTimeout     uint16
	HardTimeout     uint16
	Cookie          uint64
	PacketCount     uint64
	ByteCount       uint64
	Actions         []FlowAction
}

func (r *FlowStats) UnmarshalBinary(data []byte) error {
	if len(data) < 88 {
		return ErrInvalidPacketLength
	}

	r.length = binary.BigEndian.Uint16(data[0:2])
	if r.length < 88 {
		return ErrInvalidPacketLength
	}
	r.TableID = data[2]
	// data[3] is padding
	r.Match = &FlowMatch{}
	if err := r.Match.UnmarshalBinary(data[4:44]); err != nil {
		return err
	}
	r.DurationSec = binary.BigEndian.Uint32(data[44:48])
	r.DurationNanoSec = binary.BigEndian.Uint32(data[48:52])
	r.Priority = binary.BigEndian.Uint16(data[52:54])
	r.IdleTimeout = binary.BigEndian.Uint16(data[54:56])
	r.HardTimeout = binary.BigEndian.Uint16(data[56:58])
	// data[58:64] is padding
	r.Cookie = binary.BigEndian.Uint64(data[64:72])
	r.PacketCount = binary.BigEndian.Uint64(data[72:80])
	r.ByteCount = binary.BigEndian.Uint64(data[80:88])

	// Unmarshal actions
	remain := r.length - 88 // length will be size of action structures
	offset := 88
	if remain > 0 {
		r.Actions = make([]FlowAction, 0)
	}
	for remain > 0 {
		buf := data[offset:]
		if len(buf) < 4 {
			return ErrInvalidPacketLength
		}
		actionType := ActionType(binary.BigEndian.Uint16(buf[0:2]))
		actionLen := binary.BigEndian.Uint16(buf[2:4])
		if len(buf) < int(actionLen) || remain < actionLen {
			return ErrInvalidPacketLength
		}
		action, err := parseFlowAction(actionType, buf)
		if err != nil {
			return err
		}
		r.Actions = append(r.Actions, action)

		remain -= actionLen
		offset += int(actionLen)
	}

	return nil
}

func parseFlowAction(t ActionType, data []byte) (FlowAction, error) {
	var action FlowAction

	switch t {
	case OFPAT_OUTPUT:
		action = &FlowActionOutput{}
	case OFPAT_SET_VLAN_VID:
		action = &FlowActionSetVLANID{}
	case OFPAT_SET_VLAN_PCP:
		action = &FlowActionSetVLANPriority{}
	case OFPAT_STRIP_VLAN:
		action = &FlowActionStripVLAN{}
	case OFPAT_SET_DL_SRC:
		action = &FlowActionSetSrcMAC{}
	case OFPAT_SET_DL_DST:
		action = &FlowActionSetDstMAC{}
	case OFPAT_SET_NW_SRC:
		action = &FlowActionSetSrcIP{}
	case OFPAT_SET_NW_DST:
		action = &FlowActionSetDstIP{}
	case OFPAT_SET_NW_TOS:
		action = &FlowActionSetTOS{}
	case OFPAT_SET_TP_SRC:
		action = &FlowActionSetSrcPort{}
	case OFPAT_SET_TP_DST:
		action = &FlowActionSetDstPort{}
	case OFPAT_ENQUEUE:
		action = &FlowActionEnqueue{}
	default:
		return nil, fmt.Errorf("unsupported flow action type: %v", t)
	}

	if err := action.UnmarshalBinary(data); err != nil {
		return nil, err
	}

	return action, nil
}

type FlowStatsReplyMessage struct {
	StatsReplyMessage
	Flows []*FlowStats
}

func (r *FlowStatsReplyMessage) UnmarshalBinary(data []byte) error {
	reply := StatsReplyMessage{}
	if err := reply.UnmarshalBinary(data); err != nil {
		return err
	}
	if int(reply.Length) != len(data) {
		return ErrInvalidPacketLength
	}

	r.StatsReplyMessage = reply

	offset := 12
	buf := data[offset:]
	if len(buf) >= 88 {
		r.Flows = make([]*FlowStats, 0)
	}
	for len(buf) >= 88 {
		flow := &FlowStats{}
		if err := flow.UnmarshalBinary(buf); err != nil {
			return err
		}
		r.Flows = append(r.Flows, flow)
		offset += int(flow.length)
		if offset >= len(data) {
			break
		}
		buf = data[offset:]
	}

	return nil
}

type FlowStatsRequest struct {
	Match *FlowMatch
}

func (r *FlowStatsRequest) MarshalBinary() ([]byte, error) {
	match, err := r.Match.MarshalBinary()
	if err != nil {
		return nil, err
	}

	v := make([]byte, 44)
	copy(v[0:40], match)
	v[40] = 0xFF // for all tables
	// We don't support output port constraint
	binary.BigEndian.PutUint16(v[42:44], uint16(OFPP_NONE))

	return v, nil
}

// TODO: Implement OFPST_AGGREGATE

// TODO: Implement OFPST_TABLE

// TODO: Implement OFPST_PORT

// TODO: Implement OFPST_QUEUE
