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

package of13

import (
	"github.com/superkkt/cherry/cherryd/openflow"
)

type PortStatus struct {
	openflow.Message
	reason uint8
	port   openflow.Port
}

func (r PortStatus) Reason() openflow.PortReason {
	switch r.reason {
	case OFPPR_ADD:
		return openflow.PortAdded
	case OFPPR_DELETE:
		return openflow.PortDeleted
	case OFPPR_MODIFY:
		return openflow.PortModified
	default:
		return openflow.PortReason(r.reason)
	}
}

func (r PortStatus) Port() openflow.Port {
	return r.port
}

func (r *PortStatus) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 72 {
		return openflow.ErrInvalidPacketLength
	}
	r.reason = payload[0]
	r.port = new(Port)
	if err := r.port.UnmarshalBinary(payload[8:]); err != nil {
		return err
	}

	return nil
}
