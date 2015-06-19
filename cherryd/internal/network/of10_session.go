/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package network

import (
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"git.sds.co.kr/cherry.git/cherryd/openflow/of10"
	"git.sds.co.kr/cherry.git/cherryd/openflow/trans"
)

type of10Session struct {
	log    log.Logger
	device *Device
}

func newOF10Session(log log.Logger, d *Device) *of10Session {
	return &of10Session{
		log:    log,
		device: d,
	}
}

func (r *of10Session) OnHello(f openflow.Factory, w trans.Writer, v openflow.Hello) error {
	if err := sendHello(f, w); err != nil {
		return fmt.Errorf("failed to send HELLO: %v", err)
	}
	if err := sendSetConfig(f, w); err != nil {
		return fmt.Errorf("failed to send SET_CONFIG: %v", err)
	}
	if err := sendFeaturesRequest(f, w); err != nil {
		return fmt.Errorf("failed to send FEATURE_REQUEST: %v", err)
	}
	if err := sendBarrierRequest(f, w); err != nil {
		return fmt.Errorf("failed to send BARRIER_REQUEST: %v", err)
	}
	if err := sendRemovingAllFlows(f, w); err != nil {
		return fmt.Errorf("failed to send FLOW_MOD to remove all flows: %v", err)
	}
	if err := sendDescriptionRequest(f, w); err != nil {
		return fmt.Errorf("failed to send DESCRIPTION_REQUEST: %v", err)
	}
	if err := sendBarrierRequest(f, w); err != nil {
		return fmt.Errorf("failed to send BARRIER_REQUEST: %v", err)
	}

	return nil
}

func (r *of10Session) OnError(f openflow.Factory, w trans.Writer, v openflow.Error) error {
	return nil
}

func (r *of10Session) OnFeaturesReply(f openflow.Factory, w trans.Writer, v openflow.FeaturesReply) error {
	ports := v.Ports()
	for _, p := range ports {
		if p.Number() > of10.OFPP_MAX {
			continue
		}
		r.device.addPort(p.Number(), p)
		if !p.IsPortDown() && !p.IsLinkDown() && r.device.isValid() {
			// Send LLDP to update network topology
			if err := sendLLDP(r.device.ID(), f, w, p); err != nil {
				r.log.Err(fmt.Sprintf("failed to send LLDP: %v", err))
			}
		}
		r.log.Debug(fmt.Sprintf("Port: num=%v, AdminUp=%v, LinkUp=%v", p.Number(), !p.IsPortDown(), !p.IsLinkDown()))
	}

	return nil
}

func (r *of10Session) OnGetConfigReply(f openflow.Factory, w trans.Writer, v openflow.GetConfigReply) error {
	return nil
}

func (r *of10Session) OnDescReply(f openflow.Factory, w trans.Writer, v openflow.DescReply) error {
	return nil
}

func (r *of10Session) OnPortDescReply(f openflow.Factory, w trans.Writer, v openflow.PortDescReply) error {
	return nil
}

func (r *of10Session) OnPortStatus(f openflow.Factory, w trans.Writer, v openflow.PortStatus) error {
	return nil
}

func (r *of10Session) OnFlowRemoved(f openflow.Factory, w trans.Writer, v openflow.FlowRemoved) error {
	return nil
}

func (r *of10Session) OnPacketIn(f openflow.Factory, w trans.Writer, v openflow.PacketIn) error {
	return nil
}
