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
	"git.sds.co.kr/cherry.git/cherryd/openflow/of13"
	"io"
	"strings"
)

type OF13Controller struct {
	device *Device
	log    log.Logger
}

func NewOF13Controller(d *Device, log log.Logger) *OF13Controller {
	return &OF13Controller{
		device: d,
		log:    log,
	}
}

func (r *OF13Controller) OnHello(f openflow.Factory, w io.Writer, v openflow.Hello) error {
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
	// Make sure that the installed flows are removed before setTableMiss() is called
	if err := sendBarrierRequest(f, w); err != nil {
		return err
	}
	if err := sendDescriptionRequest(f, w); err != nil {
		return err
	}
	// Make sure that DESCRIPTION_REPLY is received before PORT_DESCRIPTION_REPLY
	if err := sendBarrierRequest(f, w); err != nil {
		return err
	}
	if err := sendPortDescriptionRequest(f, w); err != nil {
		return err
	}

	return nil
}

func (r *OF13Controller) OnError(f openflow.Factory, w io.Writer, v openflow.Error) error {
	return nil
}

func (r *OF13Controller) OnFeaturesReply(f openflow.Factory, w io.Writer, v openflow.FeaturesReply) error {
	return nil
}

func (r *OF13Controller) OnGetConfigReply(f openflow.Factory, w io.Writer, v openflow.GetConfigReply) error {
	return nil
}

func isHP2920_24G(msg openflow.DescReply) bool {
	return strings.HasPrefix(msg.Manufacturer(), "HP") && strings.HasPrefix(msg.Hardware(), "2920-24G")
}

func isAS460054_T(msg openflow.DescReply) bool {
	return strings.Contains(msg.Hardware(), "AS4600-54T")
}

func (r *OF13Controller) setTableMiss(f openflow.Factory, w io.Writer, tableID uint8, inst openflow.Instruction) error {
	match, err := f.NewMatch() // Wildcard
	if err != nil {
		return err
	}

	msg, err := f.NewFlowMod(openflow.FlowAdd)
	if err != nil {
		return err
	}
	msg.SetTableID(tableID)
	// Permanent flow entry
	msg.SetIdleTimeout(0)
	msg.SetHardTimeout(0)
	// Table-miss entry should have zero priority
	msg.SetPriority(0)
	msg.SetFlowMatch(match)
	msg.SetFlowInstruction(inst)

	return send(w, msg)
}

func (r *OF13Controller) setHP2920TableMiss(f openflow.Factory, w io.Writer) error {
	// Table-100 is a hardware table, and Table-200 is a software table
	// that has very low performance.
	r.log.Debug("Setting HP2920 TableMiss entries..")

	inst, err := f.NewInstruction()
	if err != nil {
		return err
	}

	// 0 -> 100
	inst.GotoTable(100)
	if err := r.setTableMiss(f, w, 0, inst); err != nil {
		return fmt.Errorf("failed to set table_miss flow entry: %v", err)
	}
	// 100 -> 200
	inst.GotoTable(200)
	if err := r.setTableMiss(f, w, 100, inst); err != nil {
		return fmt.Errorf("failed to set table_miss flow entry: %v", err)
	}
	// 200 -> Controller
	action, err := f.NewAction()
	if err != nil {
		return err
	}
	action.SetOutPort(openflow.OutController)
	inst.ApplyAction(action)
	if err := r.setTableMiss(f, w, 200, inst); err != nil {
		return fmt.Errorf("failed to set table_miss flow entry: %v", err)
	}
	r.device.setFlowTableID(200)

	return nil
}

func (r *OF13Controller) setAS4600TableMiss(f openflow.Factory, w io.Writer) error {
	r.log.Debug("Setting AS4600 TableMiss entries..")

	// FIXME:
	// AS460054-T gives an error (type=5, code=1) that means TABLE_FULL
	// when we install a table-miss flow on Table-0 after we delete all
	// flows already installed from the switch. Is this a bug of this switch??

	return nil
}

func (r *OF13Controller) setDefaultTableMiss(f openflow.Factory, w io.Writer) error {
	r.log.Debug("Setting default TableMiss entries..")

	inst, err := f.NewInstruction()
	if err != nil {
		return err
	}

	// 0 -> Controller
	action, err := f.NewAction()
	if err != nil {
		return err
	}
	action.SetOutPort(openflow.OutController)
	inst.ApplyAction(action)
	if err := r.setTableMiss(f, w, 0, inst); err != nil {
		return fmt.Errorf("failed to set table_miss flow entry: %v", err)
	}
	r.device.setFlowTableID(0)

	return nil
}

func (r *OF13Controller) OnDescReply(f openflow.Factory, w io.Writer, v openflow.DescReply) error {
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

func (r *OF13Controller) OnPortDescReply(f openflow.Factory, w io.Writer, v openflow.PortDescReply) error {
	ports := v.Ports()
	for _, p := range ports {
		r.log.Debug(fmt.Sprintf("Adding new port: num=%v, port=%v", p.Number(), p))
		if p.Number() > of13.OFPP_MAX {
			r.log.Debug("Ignore the port. Port number > of13.OFPP_MAX.")
			continue
		}
		r.device.addPort(p.Number(), p)
		// TODO: Send LLDP and set current timestamp
	}

	return nil
}

func (r *OF13Controller) OnPortStatus(f openflow.Factory, w io.Writer, v openflow.PortStatus) error {
	p := v.Port()
	r.log.Debug(fmt.Sprintf("Updating port: num=%v, port=%v", p.Number(), p))
	if p.Number() > of13.OFPP_MAX {
		r.log.Debug("Ignore the port. Port number > of13.OFPP_MAX.")
		return nil
	}
	r.device.updatePort(p.Number(), p)

	return nil
}

func (r *OF13Controller) OnFlowRemoved(f openflow.Factory, w io.Writer, v openflow.FlowRemoved) error {
	return nil
}

func (r *OF13Controller) OnPacketIn(f openflow.Factory, w io.Writer, v openflow.PacketIn) error {
	return nil
}
