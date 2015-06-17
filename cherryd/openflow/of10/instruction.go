/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of10

import (
	"errors"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type Instruction struct {
	err    error
	action openflow.Action
}

func (r *Instruction) Error() error {
	return r.err
}

func (r *Instruction) GotoTable(tableID uint8) {
	// OpenFlow 1.0 does not support GotoTable
}

func (r *Instruction) WriteAction(act openflow.Action) {
	if act == nil {
		panic("act is nil")
	}
	r.action = act
}

func (r *Instruction) ApplyAction(act openflow.Action) {
	if act == nil {
		panic("act is nil")
	}
	r.action = act
}

func (r *Instruction) MarshalBinary() ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}

	if r.action == nil {
		return nil, errors.New("empty action of an instruction")
	}

	return r.action.MarshalBinary()
}
