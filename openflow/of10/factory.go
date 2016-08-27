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

package of10

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/superkkt/cherry/openflow"
)

// Concrete factory
type Factory struct {
	xid uint32
}

func NewFactory() openflow.Factory {
	return &Factory{}
}

func (r *Factory) getTransactionID() uint32 {
	// Transaction ID will be started from 1, not 0.
	return atomic.AddUint32(&r.xid, 1)
}

func (r *Factory) NewHello() (openflow.Hello, error) {
	return NewHello(r.getTransactionID()), nil
}

func (r *Factory) NewEchoRequest() (openflow.EchoRequest, error) {
	return NewEchoRequest(r.getTransactionID()), nil
}

func (r *Factory) NewEchoReply() (openflow.EchoReply, error) {
	return NewEchoReply(r.getTransactionID()), nil
}

func (r *Factory) NewAction() (openflow.Action, error) {
	return NewAction(), nil
}

func (r *Factory) NewMatch() (openflow.Match, error) {
	return NewMatch(), nil
}

func (r *Factory) NewBarrierRequest() (openflow.BarrierRequest, error) {
	return NewBarrierRequest(r.getTransactionID()), nil
}

func (r *Factory) NewBarrierReply() (openflow.BarrierReply, error) {
	return new(BarrierReply), nil
}

func (r *Factory) NewSetConfig() (openflow.SetConfig, error) {
	return NewSetConfig(r.getTransactionID()), nil
}

func (r *Factory) NewGetConfigRequest() (openflow.GetConfigRequest, error) {
	return NewGetConfigRequest(r.getTransactionID()), nil
}

func (r *Factory) NewGetConfigReply() (openflow.GetConfigReply, error) {
	return new(GetConfigReply), nil
}

func (r *Factory) NewFeaturesRequest() (openflow.FeaturesRequest, error) {
	return NewFeaturesRequest(r.getTransactionID()), nil
}

func (r *Factory) NewFeaturesReply() (openflow.FeaturesReply, error) {
	return new(FeaturesReply), nil
}

func getFlowModCmd(cmd openflow.FlowModCmd) uint16 {
	var c uint16
	switch cmd {
	case openflow.FlowAdd:
		c = OFPFC_ADD
	case openflow.FlowModify:
		c = OFPFC_MODIFY
	case openflow.FlowDelete:
		c = OFPFC_DELETE
	default:
		panic(fmt.Sprintf("unexpected FlowModCmd: %v", cmd))
	}

	return c
}

func (r *Factory) NewFlowMod(cmd openflow.FlowModCmd) (openflow.FlowMod, error) {
	return NewFlowMod(r.getTransactionID(), getFlowModCmd(cmd)), nil
}

func (r *Factory) NewFlowRemoved() (openflow.FlowRemoved, error) {
	return new(FlowRemoved), nil
}

func (r *Factory) NewPacketIn() (openflow.PacketIn, error) {
	return new(PacketIn), nil
}

func (r *Factory) NewPacketOut() (openflow.PacketOut, error) {
	return NewPacketOut(r.getTransactionID()), nil
}

func (r *Factory) NewPortStatus() (openflow.PortStatus, error) {
	return new(PortStatus), nil
}

func (r *Factory) NewDescRequest() (openflow.DescRequest, error) {
	return NewDescRequest(r.getTransactionID()), nil
}

func (r *Factory) NewDescReply() (openflow.DescReply, error) {
	return new(DescReply), nil
}

func (r *Factory) NewFlowStatsRequest() (openflow.FlowStatsRequest, error) {
	return NewFlowStatsRequest(r.getTransactionID()), nil
}

// TODO: NewFlowStatsReply() (openflow.FlowStatsReply, error) {

func (r *Factory) NewPortDescRequest() (openflow.PortDescRequest, error) {
	return nil, errors.New("of10 does not support PortDescRequest")
}

func (r *Factory) NewPortDescReply() (openflow.PortDescReply, error) {
	return nil, errors.New("of10 does not support PortDescReply")
}

func (r *Factory) NewTableFeaturesRequest() (openflow.TableFeaturesRequest, error) {
	return nil, errors.New("of10 does not support TableFeaturesRequest")
}

func (r *Factory) NewError() (openflow.Error, error) {
	return new(openflow.BaseError), nil
}

// TODO: NewTableFeaturesReply() (TableFeaturesReply, error)

func (r *Factory) NewInstruction() (openflow.Instruction, error) {
	return new(Instruction), nil
}

func (r *Factory) NewQueueGetConfigRequest() (openflow.QueueGetConfigRequest, error) {
	return NewQueueGetConfigRequest(r.getTransactionID()), nil
}
