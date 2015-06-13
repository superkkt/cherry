/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package controller

import (
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"git.sds.co.kr/cherry.git/cherryd/openflow/of10"
	"io"
)

type OF10Controller struct {
	device *Device
	log    log.Logger
}

func NewOF10Controller(d *Device, log log.Logger) *OF10Controller {
	return &OF10Controller{
		device: d,
		log:    log,
	}
}

func (r *OF10Controller) OnHello(f openflow.Factory, w io.Writer, v openflow.Hello) error {
	if err := sendHello(f, w); err != nil {
		return err
	}
	if err := sendSetConfig(f, w); err != nil {
		return err
	}
	if err := sendFeaturesRequest(f, w); err != nil {
		return err
	}
	if err := sendRemovingAllFlows(f, w); err != nil {
		return err
	}
	if err := sendDescriptionRequest(f, w); err != nil {
		return err
	}
	if err := sendBarrierRequest(f, w); err != nil {
		return err
	}

	return nil
}

func (r *OF10Controller) OnError(f openflow.Factory, w io.Writer, v openflow.Error) error {
	return nil
}

func (r *OF10Controller) OnFeaturesReply(f openflow.Factory, w io.Writer, v openflow.FeaturesReply) error {
	ports := v.Ports()
	for _, p := range ports {
		r.log.Debug(fmt.Sprintf("Adding new port: num=%v, port=%v", p.Number(), p))
		if p.Number() > of10.OFPP_MAX {
			r.log.Debug("Ignore the port. Port number > of10.OFPP_MAX.")
			continue
		}
		r.device.addPort(p.Number(), p)
		// TODO: Send LLDP and set current timestamp
	}

	return nil
}

func (r *OF10Controller) OnGetConfigReply(f openflow.Factory, w io.Writer, v openflow.GetConfigReply) error {
	return nil
}

func (r *OF10Controller) OnDescReply(f openflow.Factory, w io.Writer, v openflow.DescReply) error {
	return nil
}

func (r *OF10Controller) OnPortDescReply(f openflow.Factory, w io.Writer, v openflow.PortDescReply) error {
	return nil
}

func (r *OF10Controller) OnPortStatus(f openflow.Factory, w io.Writer, v openflow.PortStatus) error {
	p := v.Port()
	r.log.Debug(fmt.Sprintf("Updating port: num=%v, port=%v", p.Number(), p))
	if p.Number() > of10.OFPP_MAX {
		r.log.Debug("Ignore the port. Port number > of10.OFPP_MAX.")
		return nil
	}
	r.device.updatePort(p.Number(), p)

	return nil
}

func (r *OF10Controller) OnFlowRemoved(f openflow.Factory, w io.Writer, v openflow.FlowRemoved) error {
	return nil
}

func (r *OF10Controller) OnPacketIn(f openflow.Factory, w io.Writer, v openflow.PacketIn) error {
	return nil
}
