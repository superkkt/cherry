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
	"errors"
)

var (
	ErrInvalidPacketLength   = errors.New("invalid packet length")
	ErrUnsupportedVersion    = errors.New("unsupported protocol version")
	ErrUnsupportedMessage    = errors.New("unsupported message type")
	ErrInvalidMACAddress     = errors.New("invalid MAC address")
	ErrInvalidIPAddress      = errors.New("invalid IP address")
	ErrUnsupportedIPProtocol = errors.New("unsupported IP protocol")
	ErrUnsupportedEtherType  = errors.New("unsupported Ethernet type")
	ErrMissingIPProtocol     = errors.New("missing IP protocol")
	ErrMissingEtherType      = errors.New("missing Ethernet type")
	ErrUnsupportedMatchType  = errors.New("unsupported flow match type")
	ErrInvalidPropertyMethod = errors.New("invalid property method")
)

// Abstract factory
type Factory interface {
	NewAction() (Action, error)
	NewBarrierRequest() (BarrierRequest, error)
	NewBarrierReply() (BarrierReply, error)
	NewDescRequest() (DescRequest, error)
	NewDescReply() (DescReply, error)
	NewEchoRequest() (EchoRequest, error)
	NewEchoReply() (EchoReply, error)
	NewError() (Error, error)
	NewFeaturesRequest() (FeaturesRequest, error)
	NewFeaturesReply() (FeaturesReply, error)
	NewFlowMod(cmd FlowModCmd) (FlowMod, error)
	NewFlowRemoved() (FlowRemoved, error)
	NewFlowStatsRequest() (FlowStatsRequest, error)
	// TODO: NewFlowStatsReply() (FlowStatsReply, error)
	NewGetConfigRequest() (GetConfigRequest, error)
	NewGetConfigReply() (GetConfigReply, error)
	NewHello() (Hello, error)
	NewInstruction() (Instruction, error)
	NewMatch() (Match, error)
	NewPacketIn() (PacketIn, error)
	NewPacketOut() (PacketOut, error)
	NewPortDescRequest() (PortDescRequest, error)
	NewPortDescReply() (PortDescReply, error)
	NewPortStatus() (PortStatus, error)
	NewQueueGetConfigRequest() (QueueGetConfigRequest, error)
	NewSetConfig() (SetConfig, error)
	NewTableFeaturesRequest() (TableFeaturesRequest, error)
	// TODO: NewTableFeaturesReply() (TableFeaturesReply, error)
}
