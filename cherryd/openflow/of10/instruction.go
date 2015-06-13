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
	action openflow.Action
}

func (r *Instruction) GotoTable(tableID uint8) error {
	// OpenFlow 1.0 does not support GotoTable
	return nil
}

func (r *Instruction) WriteAction(act openflow.Action) error {
	if act == nil {
		return errors.New("act is nil")
	}
	r.action = act

	return nil
}

func (r *Instruction) ApplyAction(act openflow.Action) error {
	if act == nil {
		return errors.New("act is nil")
	}
	r.action = act

	return nil
}

func (r *Instruction) MarshalBinary() ([]byte, error) {
	if r.action == nil {
		return nil, errors.New("empty instruction")
	}

	return r.action.MarshalBinary()
}
