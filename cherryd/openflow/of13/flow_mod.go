/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

import (
	"encoding/binary"
	"errors"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type FlowModConfig struct {
	Cookie      uint64
	CookieMask  uint64
	TableID     uint8
	IdleTimeout uint16
	HardTimeout uint16
	Priority    uint16
	Match       openflow.Match
	Action      openflow.Action
}

type FlowMod struct {
	openflow.Message
	command uint8
	config  *FlowModConfig
}

func newFlowMod(xid uint32, command uint8, config *FlowModConfig) *FlowMod {
	return &FlowMod{
		Message: openflow.NewMessage(openflow.Ver13, OFPT_FLOW_MOD, xid),
		command: command,
		config:  config,
	}
}

func NewFlowModAdd(xid uint32, config *FlowModConfig) *FlowMod {
	return newFlowMod(xid, OFPFC_ADD, config)
}

func NewFlowModModify(xid uint32, config *FlowModConfig) *FlowMod {
	return newFlowMod(xid, OFPFC_MODIFY, config)
}

func NewFlowModDelete(xid uint32, config *FlowModConfig) *FlowMod {
	return newFlowMod(xid, OFPFC_DELETE, config)
}

func marshalWriteAction(action openflow.Action) ([]byte, error) {
	v, err := action.MarshalBinary()
	if err != nil {
		return nil, err
	}

	result := make([]byte, 8)
	result = append(result, v...)
	binary.BigEndian.PutUint16(result[0:2], OFPIT_WRITE_ACTIONS)
	binary.BigEndian.PutUint16(result[2:4], uint16(len(result)))

	return result, nil
}

func (r *FlowMod) MarshalBinary() ([]byte, error) {
	if r.config.Match == nil {
		return nil, errors.New("empty flow match")
	}

	v := make([]byte, 40)
	binary.BigEndian.PutUint64(v[0:8], r.config.Cookie)
	binary.BigEndian.PutUint64(v[8:16], r.config.CookieMask)
	v[16] = r.config.TableID
	v[17] = r.command
	binary.BigEndian.PutUint16(v[18:20], r.config.IdleTimeout)
	binary.BigEndian.PutUint16(v[20:22], r.config.HardTimeout)
	binary.BigEndian.PutUint16(v[22:24], r.config.Priority)
	binary.BigEndian.PutUint32(v[24:28], OFP_NO_BUFFER)
	binary.BigEndian.PutUint32(v[28:32], OFPP_ANY)
	binary.BigEndian.PutUint32(v[32:36], OFPP_ANY)
	binary.BigEndian.PutUint16(v[36:38], OFPFF_SEND_FLOW_REM)
	// v[38:40] is padding
	match, err := r.config.Match.MarshalBinary()
	if err != nil {
		return nil, err
	}
	v = append(v, match...)
	if r.config.Action != nil {
		action, err := marshalWriteAction(r.config.Action)
		if err != nil {
			return nil, err
		}
		v = append(v, action...)
	}

	r.SetPayload(v)
	return r.Message.MarshalBinary()
}
