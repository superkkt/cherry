/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package controller

import (
	"bytes"
	"encoding"
	"errors"
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"git.sds.co.kr/cherry.git/cherryd/net/protocol"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"git.sds.co.kr/cherry.git/cherryd/openflow/trans"
	"io"
	"net"
	"strconv"
	"time"
)

// I/O timeout in seconds
const (
	readTimeout  = 10
	writeTimeout = 30
)

type Handler interface {
	trans.Handler
	SetDevice(*Device)
}

type Controller struct {
	device     *Device
	trans      *trans.Transceiver
	log        log.Logger
	handler    Handler
	negotiated bool
	watcher    Watcher
	finder     Finder
	auxID      uint8
}

func NewController(c io.ReadWriteCloser, log log.Logger, w Watcher, f Finder) *Controller {
	stream := trans.NewStream(c)
	stream.SetReadTimeout(readTimeout * time.Second)
	stream.SetWriteTimeout(writeTimeout * time.Second)

	v := new(Controller)
	v.log = log
	v.watcher = w
	v.finder = f
	v.trans = trans.NewTransceiver(stream, v)

	return v
}

func (r *Controller) OnHello(f openflow.Factory, w trans.Writer, v openflow.Hello) error {
	r.log.Debug(fmt.Sprintf("HELLO (ver=%v) is received", v.Version()))

	// Ignore duplicated HELLO messages
	if r.negotiated {
		return nil
	}
	r.negotiated = true

	switch v.Version() {
	case openflow.OF10_VERSION:
		r.handler = NewOF10Controller(r.log)
	case openflow.OF13_VERSION:
		r.handler = NewOF13Controller(r.log)
	default:
		err := errors.New(fmt.Sprintf("unsupported OpenFlow version: %v", v.Version()))
		r.log.Err(err.Error())
		return err
	}

	r.log.Debug("Calling OnHello()..")
	return r.handler.OnHello(f, w, v)
}

func (r *Controller) OnError(f openflow.Factory, w trans.Writer, v openflow.Error) error {
	r.log.Err(fmt.Sprintf("Error: class=%v, code=%v, data=%v", v.Class(), v.Code(), v.Data()))
	// Just in case
	if r.device == nil {
		return nil
	}
	return r.handler.OnError(f, w, v)
}

func (r *Controller) setDevice(d *Device) {
	r.device = d
	r.handler.SetDevice(d)
}

func (r *Controller) OnFeaturesReply(f openflow.Factory, w trans.Writer, v openflow.FeaturesReply) error {
	r.log.Debug(fmt.Sprintf("FEATURES_REPLY: DPID=%v, NumBufs=%v, NumTables=%v", v.DPID(), v.NumBuffers(), v.NumTables()))

	r.auxID = v.AuxID()
	dpid := strconv.FormatUint(v.DPID(), 10)
	device := r.finder.Device(dpid)
	if device == nil {
		device = NewDevice(dpid, r.log, r.watcher, r.finder)
		r.watcher.DeviceAdded(device)
	}
	device.addController(v.AuxID(), r)
	r.setDevice(device)
	features := Features{
		DPID:       v.DPID(),
		NumBuffers: v.NumBuffers(),
		NumTables:  v.NumTables(),
	}
	r.device.setFeatures(features)

	return r.handler.OnFeaturesReply(f, w, v)
}

func (r *Controller) OnGetConfigReply(f openflow.Factory, w trans.Writer, v openflow.GetConfigReply) error {
	if r.device == nil {
		return nil
	}

	r.log.Debug(fmt.Sprintf("GET_CONFIG_REPLY is received: %v", v))
	return r.handler.OnGetConfigReply(f, w, v)
}

func (r *Controller) OnDescReply(f openflow.Factory, w trans.Writer, v openflow.DescReply) error {
	if r.device == nil {
		return nil
	}

	r.log.Debug(fmt.Sprintf("DESC_REPLY is received: %v", v))
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

func (r *Controller) OnPortDescReply(f openflow.Factory, w trans.Writer, v openflow.PortDescReply) error {
	if r.device == nil {
		return nil
	}

	r.log.Debug(fmt.Sprintf("PORT_DESC_REPLY is received: %v", v))
	return r.handler.OnPortDescReply(f, w, v)
}

func newLLDPEtherFrame(deviceID string, port openflow.Port) ([]byte, error) {
	lldp := &protocol.LLDP{
		ChassisID: protocol.LLDPChassisID{
			SubType: 7, // Locally assigned alpha-numeric string
			Data:    []byte(deviceID),
		},
		PortID: protocol.LLDPPortID{
			SubType: 5, // Interface Name
			Data:    []byte(fmt.Sprintf("cherry/%v", port.Number())),
		},
		TTL: 120,
	}
	payload, err := lldp.MarshalBinary()
	if err != nil {
		return nil, err
	}

	ethernet := &protocol.Ethernet{
		SrcMAC: port.MAC(),
		// LLDP multicast MAC address
		DstMAC: []byte{0x01, 0x80, 0xC2, 0x00, 0x00, 0x0E},
		// LLDP ethertype
		Type:    0x88CC,
		Payload: payload,
	}
	frame, err := ethernet.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return frame, nil
}

func sendLLDP(deviceID string, f openflow.Factory, w trans.Writer, p openflow.Port) error {
	lldp, err := newLLDPEtherFrame(deviceID, p)
	if err != nil {
		return err
	}

	// Packet out to the port
	action, err := f.NewAction()
	if err != nil {
		return err
	}
	if err := action.SetOutPort(openflow.OutPort(p.Number())); err != nil {
		return err
	}

	out, err := f.NewPacketOut()
	if err != nil {
		return err
	}
	// From controller
	if err := out.SetInPort(openflow.NewInPort()); err != nil {
		return err
	}
	if err := out.SetAction(action); err != nil {
		return err
	}
	if err := out.SetData(lldp); err != nil {
		return err
	}

	// TODO: Set sent timestamp to Port structure as a LLDP timer?

	return w.Write(out)
}

func (r *Controller) OnPortStatus(f openflow.Factory, w trans.Writer, v openflow.PortStatus) error {
	if r.device == nil {
		return nil
	}

	r.log.Debug(fmt.Sprintf("PORT_STATUS is received: %v", v))
	port := v.Port()
	// Is this an enabled port?
	if !port.IsPortDown() && !port.IsLinkDown() {
		r.log.Debug("Sending LLDP..")
		// Send LLDP to update network topology
		if err := sendLLDP(r.device.ID(), f, w, port); err != nil {
			r.log.Err(fmt.Sprintf("failed to send LLDP: %v", err))
		}
		r.log.Debug("Sent LLDP..")
	}

	r.log.Debug("Calling OnPortStatus() handler..")
	err := r.handler.OnPortStatus(f, w, v)
	r.log.Debug("Calling OnPortStatus() handler is done..")
	return err
}

func (r *Controller) OnFlowRemoved(f openflow.Factory, w trans.Writer, v openflow.FlowRemoved) error {
	if r.device == nil {
		return nil
	}

	r.log.Debug(fmt.Sprintf("FLOW_REMOVED is received: %v", v))
	return r.handler.OnFlowRemoved(f, w, v)
}

func getEthernet(packet []byte) (*protocol.Ethernet, error) {
	eth := new(protocol.Ethernet)
	if err := eth.UnmarshalBinary(packet); err != nil {
		return nil, err
	}

	return eth, nil
}

func isLLDP(e *protocol.Ethernet) bool {
	return e.Type == 0x88CC
}

func getLLDP(packet []byte) (*protocol.LLDP, error) {
	lldp := new(protocol.LLDP)
	if err := lldp.UnmarshalBinary(packet); err != nil {
		return nil, err
	}

	return lldp, nil
}

func isCherryLLDP(p *protocol.LLDP) bool {
	// We sent a LLDP packet that has ChassisID.SubType=7, PortID.SubType=5,
	// and port ID starting with "cherry/".
	if p.ChassisID.SubType != 7 || p.ChassisID.Data == nil {
		// Do nothing if this packet is not the one we sent
		return false
	}
	if p.PortID.SubType != 5 || p.PortID.Data == nil {
		return false
	}
	if len(p.PortID.Data) <= 7 || !bytes.HasPrefix(p.PortID.Data, []byte("cherry/")) {
		return false
	}

	return true
}

func extractDeviceInfo(p *protocol.LLDP) (deviceID string, portNum uint32, err error) {
	if !isCherryLLDP(p) {
		return "", 0, errors.New("not found cherry LLDP packet")
	}

	deviceID = string(p.ChassisID.Data)
	// PortID.Data string consists of "cherry/" and port number
	num, err := strconv.ParseUint(string(p.PortID.Data[7:]), 10, 32)
	if err != nil {
		return "", 0, err
	}

	return deviceID, uint32(num), nil
}

func (r *Controller) findNeighborPort(deviceID string, portNum uint32) (*Port, error) {
	device := r.finder.Device(deviceID)
	if device == nil {
		return nil, fmt.Errorf("failed to find a neighbor device: id=%v", deviceID)
	}
	port := device.Port(uint(portNum))
	if port == nil {
		return nil, fmt.Errorf("failed to find a neighbor port: deviceID=%v, portNum=%v", deviceID, portNum)
	}

	return port, nil
}

func (r *Controller) handleLLDP(inPort *Port, ethernet *protocol.Ethernet) error {
	r.log.Debug("Handling LLDP..")

	lldp, err := getLLDP(ethernet.Payload)
	if err != nil {
		return err
	}
	deviceID, portNum, err := extractDeviceInfo(lldp)
	if err != nil {
		// Do nothing if this packet is not the one we sent
		r.log.Debug("Unknown LLDP is received")
		return nil
	}
	port, err := r.findNeighborPort(deviceID, portNum)
	if err != nil {
		return err
	}
	r.watcher.DeviceLinked([2]*Port{inPort, port})

	return nil
}

func (r *Controller) addNewNode(inPort *Port, mac net.HardwareAddr) error {
	node := inPort.AddNode(mac)
	r.watcher.NodeAdded(node)

	return nil
}

func (r *Controller) OnPacketIn(f openflow.Factory, w trans.Writer, v openflow.PacketIn) error {
	if r.device == nil {
		return nil
	}

	r.log.Debug(fmt.Sprintf("PACKET_IN is received: %v", v))
	ethernet, err := getEthernet(v.Data())
	if err != nil {
		return err
	}
	inPort := r.device.Port(uint(v.InPort()))
	if inPort == nil {
		return fmt.Errorf("failed to find a port: deviceID=%v, portNum=%v", r.device.ID(), v.InPort())
	}

	// Process LLDP, and then add an edge among two switches
	if isLLDP(ethernet) {
		return r.handleLLDP(inPort, ethernet)
	}

	// Do we know packet sender?
	if r.finder.Node(ethernet.SrcMAC) == nil {
		// MAC learning
		if err := r.addNewNode(inPort, ethernet.SrcMAC); err != nil {
			return err
		}
	}

	// TODO: Check LLDP timer activated in sendLLDP()

	// Do nothing if the ingress port is an edge between switches and is disabled by STP.
	if r.finder.IsDisabledPort(inPort) {
		r.log.Debug(fmt.Sprintf("Ignore PACKET_IN %v:%v due to STP", r.device.ID(), v.InPort()))
		return nil
	}

	if err := r.handler.OnPacketIn(f, w, v); err != nil {
		return err
	}

	// TODO: Call packet watcher

	return nil
}

// TODO: Use context to shutdown running controllers
func (r *Controller) Run() {
	r.log.Debug("Starting a transceiver..")
	if err := r.trans.Run(); err != nil {
		r.log.Err(fmt.Sprintf("Transceiver terminated: %v", err))
	}
	r.log.Debug("Closing the transceiver..")
	r.trans.Close()
	if r.device != nil {
		r.device.removeController(r.auxID)
	}
}

func (r *Controller) SendMessage(msg encoding.BinaryMarshaler) error {
	return r.trans.Write(msg)
}

func sendHello(f openflow.Factory, w trans.Writer) error {
	msg, err := f.NewHello()
	if err != nil {
		return err
	}

	return w.Write(msg)
}

func sendSetConfig(f openflow.Factory, w trans.Writer) error {
	msg, err := f.NewSetConfig()
	if err != nil {
		return err
	}
	msg.SetFlags(openflow.FragNormal)
	msg.SetMissSendLength(0xFFFF)

	return w.Write(msg)
}

func sendFeaturesRequest(f openflow.Factory, w trans.Writer) error {
	msg, err := f.NewFeaturesRequest()
	if err != nil {
		return err
	}

	return w.Write(msg)
}

func sendDescriptionRequest(f openflow.Factory, w trans.Writer) error {
	msg, err := f.NewDescRequest()
	if err != nil {
		return err
	}

	return w.Write(msg)
}

func sendBarrierRequest(f openflow.Factory, w trans.Writer) error {
	msg, err := f.NewBarrierRequest()
	if err != nil {
		return err
	}

	return w.Write(msg)
}

func sendPortDescriptionRequest(f openflow.Factory, w trans.Writer) error {
	msg, err := f.NewPortDescRequest()
	if err != nil {
		return err
	}

	return w.Write(msg)
}

func sendRemovingAllFlows(f openflow.Factory, w trans.Writer) error {
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

	return w.Write(msg)
}
