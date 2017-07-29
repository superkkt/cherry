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

package network

import (
	"github.com/superkkt/cherry/openflow"
	"github.com/superkkt/cherry/openflow/of10"
	"github.com/superkkt/cherry/openflow/transceiver"

	"github.com/pkg/errors"
)

type of10Session struct {
	device *Device
}

func newOF10Session(d *Device) *of10Session {
	return &of10Session{
		device: d,
	}
}

func (r *of10Session) OnHello(f openflow.Factory, w transceiver.Writer, v openflow.Hello) error {
	if err := sendHello(f, w); err != nil {
		return errors.Wrap(err, "failed to send HELLO")
	}
	if err := sendSetConfig(f, w); err != nil {
		return errors.Wrap(err, "failed to send SET_CONFIG")
	}
	if err := sendFeaturesRequest(f, w); err != nil {
		return errors.Wrap(err, "failed to send FEATURE_REQUEST")
	}
	if err := sendBarrierRequest(f, w); err != nil {
		return errors.Wrap(err, "failed to send BARRIER_REQUEST")
	}
	if err := sendRemovingAllFlows(f, w); err != nil {
		return errors.Wrap(err, "failed to send FLOW_MOD to remove all flows")
	}
	if err := sendDescriptionRequest(f, w); err != nil {
		return errors.Wrap(err, "failed to send DESCRIPTION_REQUEST")
	}
	if err := sendBarrierRequest(f, w); err != nil {
		return errors.Wrap(err, "failed to send BARRIER_REQUEST")
	}
	if err := setARPSender(f, w); err != nil {
		return errors.Wrap(err, "failed to set ARP sender flow")
	}

	return nil
}

func (r *of10Session) OnError(f openflow.Factory, w transceiver.Writer, v openflow.Error) error {
	return nil
}

func (r *of10Session) OnFeaturesReply(f openflow.Factory, w transceiver.Writer, v openflow.FeaturesReply) error {
	ports := v.Ports()
	for _, p := range ports {
		logger.Debugf("PortNum=%v, AdminUp=%v, LinkUp=%v", p.Number(), !p.IsPortDown(), !p.IsLinkDown())

		if p.Number() > of10.OFPP_MAX {
			logger.Debugf("invalid port number: %v", p.Number())
			continue
		}

		r.device.setPort(p.Number(), p)

		if !p.IsPortDown() && !p.IsLinkDown() && r.device.isValid() {
			// Send LLDP to update network topology
			if err := sendLLDP(r.device, p); err != nil {
				logger.Errorf("failed to send LLDP: %v", err)
				continue
			}
			logger.Debugf("sent a LLDP packet to %v:%v", r.device.ID(), p.Number())
		}
	}

	return nil
}

func (r *of10Session) OnGetConfigReply(f openflow.Factory, w transceiver.Writer, v openflow.GetConfigReply) error {
	return nil
}

func (r *of10Session) OnDescReply(f openflow.Factory, w transceiver.Writer, v openflow.DescReply) error {
	return nil
}

func (r *of10Session) OnPortDescReply(f openflow.Factory, w transceiver.Writer, v openflow.PortDescReply) error {
	// Do nothing because OpenFlow 1.0 uses FeaturesReply instead of PortDescReply.
	return nil
}

func (r *of10Session) OnPortStatus(f openflow.Factory, w transceiver.Writer, v openflow.PortStatus) error {
	return nil
}

func (r *of10Session) OnFlowRemoved(f openflow.Factory, w transceiver.Writer, v openflow.FlowRemoved) error {
	return nil
}

func (r *of10Session) OnPacketIn(f openflow.Factory, w transceiver.Writer, v openflow.PacketIn) error {
	return nil
}
