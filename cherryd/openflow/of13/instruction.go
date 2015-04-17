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

type Instruction interface {
	encoding.BinaryMarshaler
}

type GotoTable struct {
	TableID uint8
}

func (r *GotoTable) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	binary.BigEndian.PutUint16(v[0:2], OFPIT_GOTO_TABLE)
	binary.BigEndian.PutUint16(v[2:4], 8)
	v[4] = r.TableID
	// v[5:8] is padding

	return v, nil
}

type WriteAction struct {
	Action openflow.Action
}

func (r *WriteAction) MarshalBinary() ([]byte, error) {
	if r.Action == nil {
		return nil, errors.New("empty action")
	}

	action, err := r.Action.MarshalBinary()
	if err != nil {
		return nil, err
	}

	v := make([]byte, 8)
	v = append(v, action...)
	binary.BigEndian.PutUint16(v[0:2], OFPIT_WRITE_ACTIONS)
	binary.BigEndian.PutUint16(v[2:4], uint16(len(v)))

	return v, nil
}

type ApplyAction struct {
	Action openflow.Action
}

func (r *ApplyAction) MarshalBinary() ([]byte, error) {
	if r.Action == nil {
		return nil, errors.New("empty action")
	}

	action, err := r.Action.MarshalBinary()
	if err != nil {
		return nil, err
	}

	v := make([]byte, 8)
	v = append(v, action...)
	binary.BigEndian.PutUint16(v[0:2], OFPIT_APPLY_ACTIONS)
	binary.BigEndian.PutUint16(v[2:4], uint16(len(v)))

	return v, nil
}
