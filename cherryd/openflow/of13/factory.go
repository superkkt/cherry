/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

import (
	//"errors"
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"sync/atomic"
)

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

func getFlowModCmd(cmd openflow.FlowModCmd) uint8 {
	var c uint8
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
	return NewPortDescRequest(r.getTransactionID()), nil
}

func (r *Factory) NewPortDescReply() (openflow.PortDescReply, error) {
	return new(PortDescReply), nil
}

func (r *Factory) NewTableFeaturesRequest() (openflow.TableFeaturesRequest, error) {
	return NewTableFeaturesRequest(r.getTransactionID()), nil
}

func (r *Factory) NewError() (openflow.Error, error) {
	return new(openflow.BaseError), nil
}

// TODO: NewTableFeaturesReply() (TableFeaturesReply, error)
