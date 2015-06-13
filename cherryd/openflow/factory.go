/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
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
	NewSetConfig() (SetConfig, error)
	NewTableFeaturesRequest() (TableFeaturesRequest, error)
	// TODO: NewTableFeaturesReply() (TableFeaturesReply, error)
}
