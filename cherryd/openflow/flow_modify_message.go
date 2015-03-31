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
)

type FlowModifyFlag struct {
	SendFlowRemoved bool /* Send flow removed message when flow expires or is deleted. */
	CheckOverlap    bool /* Check for overlapping entries first. */
	Emergency       bool /* Remark this is for emergency. */
}

func (r *FlowModifyFlag) MarshalBinary() ([]byte, error) {
	var v uint16 = 0

	if r.SendFlowRemoved {
		v = v | OFPFF_SEND_FLOW_REM
	}
	if r.CheckOverlap {
		v = v | OFPFF_CHECK_OVERLAP
	}
	if r.Emergency {
		v = v | OFPFF_EMERG
	}

	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, v)

	return data, nil
}

type FlowModifyMessage struct {
	header      Header
	Match       *FlowMatch
	Cookie      uint64
	Command     FlowModifyCmd
	IdleTimeout uint16
	HardTimeout uint16
	Priority    uint16
	bufferID    uint32
	port        PortNumber
	Flags       FlowModifyFlag
	Actions     []FlowAction
}

func (r *FlowModifyMessage) MarshalBinary() ([]byte, error) {
	actions := make([]byte, 0)
	for _, act := range r.Actions {
		buf, err := act.MarshalBinary()
		if err != nil {
			return nil, err
		}
		actions = append(actions, buf...)
	}
	if len(actions) > 0xFFFF-72 {
		return nil, errors.New("too many flow modification actions")
	}

	r.header.Length = 72 + uint16(len(actions))
	header, err := r.header.MarshalBinary()
	if err != nil {
		return nil, err
	}
	match, err := r.Match.MarshalBinary()
	if err != nil {
		return nil, err
	}

	v := make([]byte, r.header.Length)
	copy(v[0:8], header)
	copy(v[8:48], match)
	binary.BigEndian.PutUint64(v[48:56], r.Cookie)
	binary.BigEndian.PutUint16(v[56:58], uint16(r.Command))
	binary.BigEndian.PutUint16(v[58:60], r.IdleTimeout)
	binary.BigEndian.PutUint16(v[60:62], r.HardTimeout)
	binary.BigEndian.PutUint16(v[62:64], r.Priority)
	// We don't support buffer id
	r.bufferID = 0xFFFFFFFF
	binary.BigEndian.PutUint32(v[64:68], r.bufferID)
	// We don't support output port constraint
	r.port = OFPP_NONE
	binary.BigEndian.PutUint16(v[68:70], uint16(r.port))
	flags, err := r.Flags.MarshalBinary()
	if err != nil {
		return nil, err
	}
	copy(v[70:72], flags)
	if len(actions) > 0 {
		copy(v[72:], actions)
	}

	return v, nil
}
