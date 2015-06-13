/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package controller

import (
	"encoding"
	"errors"
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"git.sds.co.kr/cherry.git/cherryd/openflow/trans"
	"io"
	"time"
)

const (
	// I/O timeout in seconds
	ReadTimeout  = 10
	WriteTimeout = 30
)

type Controller struct {
	device     *Device
	trans      *trans.Transceiver
	log        log.Logger
	handler    trans.Handler
	negotiated bool
}

func NewController(d *Device, c io.ReadWriteCloser, log log.Logger) *Controller {
	stream := trans.NewStream(c)
	stream.SetReadTimeout(ReadTimeout * time.Second)
	stream.SetWriteTimeout(WriteTimeout * time.Second)

	v := new(Controller)
	v.device = d
	v.trans = trans.NewTransceiver(stream, v)
	v.log = log

	return v
}

func (r *Controller) OnHello(f openflow.Factory, w io.Writer, v openflow.Hello) error {
	r.log.Debug(fmt.Sprintf("HELLO (ver=%v) is received", v.Version()))

	// Ignore duplicated HELLO messages
	if r.negotiated {
		return nil
	}

	switch v.Version() {
	case openflow.OF10_VERSION:
		r.handler = NewOF10Controller(r.device, r.log)
	case openflow.OF13_VERSION:
		r.handler = NewOF13Controller(r.device, r.log)
	default:
		err := errors.New(fmt.Sprintf("unsupported OpenFlow version: %v", v.Version()))
		r.log.Err(err.Error())
		return err
	}
	r.negotiated = true

	return r.handler.OnHello(f, w, v)
}

func (r *Controller) OnError(f openflow.Factory, w io.Writer, v openflow.Error) error {
	r.log.Err(fmt.Sprintf("Error: class=%v, code=%v, data=%v", v.Class(), v.Code(), v.Data()))
	return r.handler.OnError(f, w, v)
}

func (r *Controller) OnFeaturesReply(f openflow.Factory, w io.Writer, v openflow.FeaturesReply) error {
	r.log.Debug(fmt.Sprintf("FEATURES_REPLY: DPID=%v, NumBufs=%v, NumTables=%v", v.DPID(), v.NumBuffers(), v.NumTables()))

	features := Features{
		DPID:       v.DPID(),
		NumBuffers: v.NumBuffers(),
		NumTables:  v.NumTables(),
	}
	r.device.setFeatures(features)

	return r.handler.OnFeaturesReply(f, w, v)
}

func (r *Controller) OnGetConfigReply(f openflow.Factory, w io.Writer, v openflow.GetConfigReply) error {
	return r.handler.OnGetConfigReply(f, w, v)
}

func (r *Controller) OnDescReply(f openflow.Factory, w io.Writer, v openflow.DescReply) error {
	desc := Descriptions{
		Manufacturer: v.Manufacturer(),
		Hardware:     v.Hardware(),
		Software:     v.Software(),
		Serial:       v.Serial(),
		Description:  v.Description(),
	}
	r.device.setDescriptions(desc)

	return r.handler.OnDescReply(f, w, v)
}

func (r *Controller) OnPortDescReply(f openflow.Factory, w io.Writer, v openflow.PortDescReply) error {
	return r.handler.OnPortDescReply(f, w, v)
}

func (r *Controller) OnPortStatus(f openflow.Factory, w io.Writer, v openflow.PortStatus) error {
	return r.handler.OnPortStatus(f, w, v)
}

func (r *Controller) OnFlowRemoved(f openflow.Factory, w io.Writer, v openflow.FlowRemoved) error {
	return r.handler.OnFlowRemoved(f, w, v)
}

func (r *Controller) OnPacketIn(f openflow.Factory, w io.Writer, v openflow.PacketIn) error {
	// TODO: Process LLDP, and then add an edge among two switches

	// TODO: MAC learning. Add a node if we don't have the source MAC address of this packet in our host database.

	// TODO: Do nothing if LLDPs we sent are still exploring network topology.

	// TODO: Do nothing if the ingress port is an edge between switches and is disabled by STP.

	if err := r.handler.OnPacketIn(f, w, v); err != nil {
		return err
	}

	// TODO: Call packet watcher

	return nil
}

// TODO: Use context to shutdown running controllers
func (r *Controller) Run() {
	if err := r.trans.Run(); err != nil {
		r.log.Err(err.Error())
		return
	}
}

func send(w io.Writer, msg encoding.BinaryMarshaler) error {
	p, err := msg.MarshalBinary()
	if err != nil {
		return err
	}
	_, err = w.Write(p)
	if err != nil {
		return err
	}

	return nil
}

func sendHello(f openflow.Factory, w io.Writer) error {
	msg, err := f.NewHello()
	if err != nil {
		return err
	}

	return send(w, msg)
}

func sendSetConfig(f openflow.Factory, w io.Writer) error {
	msg, err := f.NewSetConfig()
	if err != nil {
		return err
	}
	msg.SetFlags(openflow.FragNormal)
	msg.SetMissSendLength(0xFFFF)

	return send(w, msg)
}

func sendFeaturesRequest(f openflow.Factory, w io.Writer) error {
	msg, err := f.NewFeaturesRequest()
	if err != nil {
		return err
	}

	return send(w, msg)
}

func sendDescriptionRequest(f openflow.Factory, w io.Writer) error {
	msg, err := f.NewDescRequest()
	if err != nil {
		return err
	}

	return send(w, msg)
}

func sendBarrierRequest(f openflow.Factory, w io.Writer) error {
	msg, err := f.NewBarrierRequest()
	if err != nil {
		return err
	}

	return send(w, msg)
}

func sendPortDescriptionRequest(f openflow.Factory, w io.Writer) error {
	msg, err := f.NewPortDescRequest()
	if err != nil {
		return err
	}

	return send(w, msg)
}

func sendRemovingAllFlows(f openflow.Factory, w io.Writer) error {
	match, err := f.NewMatch() // Wildcard
	if err != nil {
		return err
	}

	msg, err := f.NewFlowMod(openflow.FlowDelete)
	if err != nil {
		return err
	}
	msg.SetTableID(0xFF) // Wildcard
	msg.SetFlowMatch(match)

	return send(w, msg)
}
