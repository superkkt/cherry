/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service, Inc. All rights reserved.
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

package openflow

import (
	"encoding"
)

type Echo interface {
	Data() []byte
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	// Error() returns last error message
	Error() error
	Header
	SetData(data []byte)
}

type EchoRequest interface {
	Echo
}

type EchoReply interface {
	Echo
}

type BaseEcho struct {
	err error
	Message
	data []byte
}

func (r *BaseEcho) Data() []byte {
	return r.data
}

func (r *BaseEcho) SetData(data []byte) {
	if data == nil {
		panic("data is nil")
	}
	r.data = data
}

func (r *BaseEcho) Error() error {
	return r.err
}

func (r *BaseEcho) MarshalBinary() ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}

	r.SetPayload(r.data)
	return r.Message.MarshalBinary()
}

func (r *BaseEcho) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}
	r.data = r.Payload()

	return nil
}
