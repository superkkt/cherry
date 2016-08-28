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
	"strings"

	"github.com/superkkt/cherry/openflow"
	"github.com/superkkt/cherry/openflow/of13"
	"github.com/superkkt/cherry/openflow/transceiver"

	"github.com/pkg/errors"
)

type of13Session struct {
	device *Device
}

func newOF13Session(d *Device) *of13Session {
	return &of13Session{
		device: d,
	}
}

func (r *of13Session) OnHello(f openflow.Factory, w transceiver.Writer, v openflow.Hello) error {
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
	// Make sure that the installed flows are removed before setTableMiss() is called
	if err := sendBarrierRequest(f, w); err != nil {
		return errors.Wrap(err, "failed to send BARRIER_REQUEST")
	}
	if err := setARPSender(f, w); err != nil {
		return errors.Wrap(err, "failed to set ARP sender flow")
	}
	if err := sendDescriptionRequest(f, w); err != nil {
		return errors.Wrap(err, "failed to send DESCRIPTION_REQUEST")
	}
	// Make sure that DESCRIPTION_REPLY is received before PORT_DESCRIPTION_REPLY
	if err := sendBarrierRequest(f, w); err != nil {
		return errors.Wrap(err, "failed to send BARRIER_REQUEST")
	}
	if err := sendPortDescriptionRequest(f, w); err != nil {
		return errors.Wrap(err, "failed to send DESCRIPTION_REQUEST")
	}

	return nil
}

func (r *of13Session) OnError(f openflow.Factory, w transceiver.Writer, v openflow.Error) error {
	return nil
}

func (r *of13Session) OnFeaturesReply(f openflow.Factory, w transceiver.Writer, v openflow.FeaturesReply) error {
	return nil
}

func (r *of13Session) OnGetConfigReply(f openflow.Factory, w transceiver.Writer, v openflow.GetConfigReply) error {
	return nil
}

func isHP2920_24G(msg openflow.DescReply) bool {
	return strings.HasPrefix(msg.Manufacturer(), "HP") && strings.HasPrefix(msg.Hardware(), "2920-24G")
}

func isAS460054_T(msg openflow.DescReply) bool {
	return strings.Contains(msg.Hardware(), "AS4600-54T")
}

func (r *of13Session) setTableMiss(f openflow.Factory, w transceiver.Writer, tableID uint8, inst openflow.Instruction) error {
	match, err := f.NewMatch() // Wildcard
	if err != nil {
		return err
	}

	msg, err := f.NewFlowMod(openflow.FlowAdd)
	if err != nil {
		return err
	}
	// We use MSB to represent whether the flow is table miss or not
	msg.SetCookie(0x1 << 63)
	msg.SetTableID(tableID)
	// Permanent flow entry
	msg.SetIdleTimeout(0)
	msg.SetHardTimeout(0)
	// Table-miss entry should have zero priority
	msg.SetPriority(0)
	msg.SetFlowMatch(match)
	msg.SetFlowInstruction(inst)

	return w.Write(msg)
}

func (r *of13Session) setHP2920TableMiss(f openflow.Factory, w transceiver.Writer) error {
	// Table-100 is a hardware table, and Table-200 is a software table
	// that has very low performance.
	inst, err := f.NewInstruction()
	if err != nil {
		return err
	}

	// 0 -> 100
	inst.GotoTable(100)
	if err := r.setTableMiss(f, w, 0, inst); err != nil {
		return errors.Wrap(err, "failed to set table_miss flow entry")
	}
	// 100 -> 200
	inst.GotoTable(200)
	if err := r.setTableMiss(f, w, 100, inst); err != nil {
		return errors.Wrap(err, "failed to set table_miss flow entry")
	}

	// 200 -> Controller
	outPort := openflow.NewOutPort()
	outPort.SetController()
	action, err := f.NewAction()
	if err != nil {
		return err
	}
	action.SetOutPort(outPort)

	inst.ApplyAction(action)
	if err := r.setTableMiss(f, w, 200, inst); err != nil {
		return errors.Wrap(err, "failed to set table_miss flow entry")
	}
	r.device.setFlowTableID(200)

	return nil
}

func (r *of13Session) setAS4600TableMiss(f openflow.Factory, w transceiver.Writer) error {
	// FIXME:
	// AS460054-T gives an error (type=5, code=1) that means TABLE_FULL
	// when we install a table-miss flow on Table-0 after we delete all
	// flows already installed from the switch. Is this a bug of this switch??

	return nil
}

func (r *of13Session) setDefaultTableMiss(f openflow.Factory, w transceiver.Writer) error {
	inst, err := f.NewInstruction()
	if err != nil {
		return err
	}

	// 0 -> Controller
	outPort := openflow.NewOutPort()
	outPort.SetController()
	action, err := f.NewAction()
	if err != nil {
		return err
	}
	action.SetOutPort(outPort)

	inst.ApplyAction(action)
	if err := r.setTableMiss(f, w, 0, inst); err != nil {
		return errors.Wrap(err, "failed to set table_miss flow entry")
	}
	r.device.setFlowTableID(0)

	return nil
}

func (r *of13Session) OnDescReply(f openflow.Factory, w transceiver.Writer, v openflow.DescReply) error {
	var err error

	// FIXME:
	// Implement general routines for various table structures of OF1.3 switches
	// based on table features reply
	switch {
	case isHP2920_24G(v):
		err = r.setHP2920TableMiss(f, w)
	case isAS460054_T(v):
		err = r.setAS4600TableMiss(f, w)
	default:
		err = r.setDefaultTableMiss(f, w)
	}

	return err
}

func (r *of13Session) OnPortDescReply(f openflow.Factory, w transceiver.Writer, v openflow.PortDescReply) error {
	ports := v.Ports()
	for _, p := range ports {
		if p.Number() > of13.OFPP_MAX {
			continue
		}
		r.device.addPort(p.Number(), p)
		if !p.IsPortDown() && !p.IsLinkDown() && r.device.isValid() {
			// Send LLDP to update network topology
			if err := sendLLDP(r.device.ID(), f, w, p); err != nil {
				logger.Errorf("failed to send LLDP: %v", err)
			}
		}
		logger.Debugf("PortNum=%v, AdminUp=%v, LinkUp=%v", p.Number(), !p.IsPortDown(), !p.IsLinkDown())

		if err := sendQueueConfigRequest(f, w, p.Number()); err != nil {
			logger.Errorf("failed to send the queue config request: %v", err)
		}
	}

	return nil
}

func (r *of13Session) OnPortStatus(f openflow.Factory, w transceiver.Writer, v openflow.PortStatus) error {
	return nil
}

func (r *of13Session) OnFlowRemoved(f openflow.Factory, w transceiver.Writer, v openflow.FlowRemoved) error {
	return nil
}

func (r *of13Session) OnPacketIn(f openflow.Factory, w transceiver.Writer, v openflow.PacketIn) error {
	return nil
}
