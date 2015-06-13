/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

import (
	"encoding"
	"encoding/binary"
	"errors"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type Instruction struct {
	value encoding.BinaryMarshaler
}

type gotoTable struct {
	tableID uint8
}

func (r gotoTable) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	binary.BigEndian.PutUint16(v[0:2], OFPIT_GOTO_TABLE)
	binary.BigEndian.PutUint16(v[2:4], 8)
	v[4] = r.tableID
	// v[5:8] is padding

	return v, nil
}

type writeAction struct {
	action openflow.Action
}

func (r writeAction) MarshalBinary() ([]byte, error) {
	if r.action == nil {
		return nil, errors.New("empty action")
	}

	action, err := r.action.MarshalBinary()
	if err != nil {
		return nil, err
	}

	v := make([]byte, 8)
	v = append(v, action...)
	binary.BigEndian.PutUint16(v[0:2], OFPIT_WRITE_ACTIONS)
	binary.BigEndian.PutUint16(v[2:4], uint16(len(v)))

	return v, nil
}

type applyAction struct {
	action openflow.Action
}

func (r applyAction) MarshalBinary() ([]byte, error) {
	if r.action == nil {
		return nil, errors.New("empty action")
	}

	action, err := r.action.MarshalBinary()
	if err != nil {
		return nil, err
	}

	v := make([]byte, 8)
	v = append(v, action...)
	binary.BigEndian.PutUint16(v[0:2], OFPIT_APPLY_ACTIONS)
	binary.BigEndian.PutUint16(v[2:4], uint16(len(v)))

	return v, nil
}

func (r *Instruction) GotoTable(tableID uint8) error {
	r.value = gotoTable{tableID: tableID}
	return nil
}

func (r *Instruction) WriteAction(act openflow.Action) error {
	if act == nil {
		return errors.New("act is nil")
	}
	r.value = writeAction{action: act}

	return nil
}

func (r *Instruction) ApplyAction(act openflow.Action) error {
	if act == nil {
		return errors.New("act is nil")
	}
	r.value = applyAction{action: act}

	return nil
}

func (r *Instruction) MarshalBinary() ([]byte, error) {
	if r.value == nil {
		return nil, errors.New("empty instruction")
	}

	return r.value.MarshalBinary()
}
