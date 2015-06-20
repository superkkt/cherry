/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along
 * with this program; if not, write to the Free Software Foundation, Inc.,
 * 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
 */

package of10

import (
	"errors"
	"github.com/superkkt/cherry/cherryd/openflow"
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
