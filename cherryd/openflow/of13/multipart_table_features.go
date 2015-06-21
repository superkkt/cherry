/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service 
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
	"encoding/binary"
	"github.com/superkkt/cherry/cherryd/openflow"
)

type TableFeaturesRequest struct {
	openflow.Message
}

func NewTableFeaturesRequest(xid uint32) openflow.TableFeaturesRequest {
	return &TableFeaturesRequest{
		Message: openflow.NewMessage(openflow.OF13_VERSION, OFPT_MULTIPART_REQUEST, xid),
	}
}

func (r *TableFeaturesRequest) MarshalBinary() ([]byte, error) {
	v := make([]byte, 8)
	// Table features request
	binary.BigEndian.PutUint16(v[0:2], OFPMP_TABLE_FEATURES)
	// No flags and body
	r.SetPayload(v)

	return r.Message.MarshalBinary()
}

// TODO: Implement TableFeaturesReply
