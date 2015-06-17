/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package session

import (
	"bytes"
	"encoding"
	"errors"
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"git.sds.co.kr/cherry.git/cherryd/internal/network"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"git.sds.co.kr/cherry.git/cherryd/openflow/of10"
	"git.sds.co.kr/cherry.git/cherryd/openflow/of13"
	"git.sds.co.kr/cherry.git/cherryd/openflow/trans"
	"git.sds.co.kr/cherry.git/cherryd/protocol"
	"io"
	"net"
	"strconv"
)

type processor interface {
	ProcessPacket(network.Finder, *protocol.Ethernet, *network.Port) error
	ProcessPortChange(network.Finder, *network.Device, openflow.PortStatus) error
	ProcessDeviceClose(network.Finder, *network.Device) error
}

type sessionHandler interface {
	trans.Handler
	setDevice(*network.Device)
}

type Controller struct {
	device     *network.Device
	trans      *trans.Transceiver
	log        log.Logger
	handler    sessionHandler
	negotiated bool
	watcher    network.Watcher
	finder     network.Finder
	processor  processor
	auxID      uint8
}

type Config struct {
	Conn      io.ReadWriteCloser
	Logger    log.Logger
	Watcher   network.Watcher
	Finder    network.Finder
	Processor processor
}

func NewController(c Config) *Controller {
	stream := trans.NewStream(c.Conn)

	v := new(Controller)
	v.log = c.Logger
	v.watcher = c.Watcher
	v.finder = c.Finder
	v.processor = c.Processor
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

func (r *Controller) setDevice(version uint8, device *network.Device, features network.Features) {
	r.device = device
	r.handler.setDevice(device)
	r.device.SetFeatures(features)
	switch version {
	case openflow.OF10_VERSION:
		r.device.SetFactory(of10.NewFactory())
	case openflow.OF13_VERSION:
		r.device.SetFactory(of13.NewFactory())
	default:
		panic("Unsupported OpenFlow version")
	}
}

func (r *Controller) OnFeaturesReply(f openflow.Factory, w trans.Writer, v openflow.FeaturesReply) error {
	r.log.Debug(fmt.Sprintf("FEATURES_REPLY: DPID=%v, NumBufs=%v, NumTables=%v", v.DPID(), v.NumBuffers(), v.NumTables()))

	r.auxID = v.AuxID()
	dpid := strconv.FormatUint(v.DPID(), 10)
	device := r.finder.Device(dpid)
	if device == nil {
		device = network.NewDevice(dpid, r.log, r.watcher, r.finder)
		r.watcher.DeviceAdded(device)
	}
	device.AddController(v.AuxID(), r)

	features := network.Features{
		DPID:       v.DPID(),
		NumBuffers: v.NumBuffers(),
		NumTables:  v.NumTables(),
	}
	r.setDevice(v.Version(), device, features)

	return r.handler.OnFeaturesReply(f, w, v)
}

func (r *Controller) OnGetConfigReply(f openflow.Factory, w trans.Writer, v openflow.GetConfigReply) error {
	r.log.Debug("GET_CONFIG_REPLY is received")

	if r.device == nil {
		r.log.Warning("Uninitialized device!")
		return nil
	}

	return r.handler.OnGetConfigReply(f, w, v)
}

func (r *Controller) OnDescReply(f openflow.Factory, w trans.Writer, v openflow.DescReply) error {
	r.log.Debug("DESC_REPLY is received")
	r.log.Debug(fmt.Sprintf("Manufacturer=%v", v.Manufacturer()))
	r.log.Debug(fmt.Sprintf("Hardware=%v", v.Hardware()))
	r.log.Debug(fmt.Sprintf("Software=%v", v.Software()))
	r.log.Debug(fmt.Sprintf("Serial=%v", v.Serial()))
	r.log.Debug(fmt.Sprintf("Description=%v", v.Description()))

	if r.device == nil {
		r.log.Warning("Uninitialized device!")
		return nil
	}

	desc := network.Descriptions{
		Manufacturer: v.Manufacturer(),
		Hardware:     v.Hardware(),
		Software:     v.Software(),
		Serial:       v.Serial(),
		Description:  v.Description(),
	}
	r.device.SetDescriptions(desc)

	return r.handler.OnDescReply(f, w, v)
}

func (r *Controller) OnPortDescReply(f openflow.Factory, w trans.Writer, v openflow.PortDescReply) error {
	r.log.Debug(fmt.Sprintf("PORT_DESC_REPLY is received: %v ports", len(v.Ports())))

	if r.device == nil {
		r.log.Warning("Uninitialized device!")
		return nil
	}

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

	outPort := openflow.NewOutPort()
	outPort.SetValue(p.Number())

	// Packet out to the port
	action, err := f.NewAction()
	if err != nil {
		return err
	}
	action.SetOutPort(outPort)

	out, err := f.NewPacketOut()
	if err != nil {
		return err
	}
	// From controller
	out.SetInPort(openflow.NewInPort())
	out.SetAction(action)
	out.SetData(lldp)

	return w.Write(out)
}

func (r *Controller) OnPortStatus(f openflow.Factory, w trans.Writer, v openflow.PortStatus) error {
	r.log.Debug("PORT_STATUS is received")

	if r.device == nil {
		r.log.Warning("Uninitialized device!")
		return nil
	}

	port := v.Port()
	r.log.Debug(fmt.Sprintf("Device=%v, PortNum=%v: AdminUp=%v, LinkUp=%v", r.device.ID(), port.Number(), !port.IsPortDown(), !port.IsLinkDown()))
	if err := r.handler.OnPortStatus(f, w, v); err != nil {
		return err
	}
	if err := r.processor.ProcessPortChange(r.finder, r.device, v); err != nil {
		return err
	}

	// Is this an enabled port?
	if !port.IsPortDown() && !port.IsLinkDown() {
		// Send LLDP to update network topology
		if err := sendLLDP(r.device.ID(), f, w, port); err != nil {
			return fmt.Errorf("failed to send LLDP: %v", err)
		}
	} else {
		// Send port removed event
		p := r.device.Port(port.Number())
		if p != nil {
			r.watcher.PortRemoved(p)
		}
	}

	return nil
}

func (r *Controller) OnFlowRemoved(f openflow.Factory, w trans.Writer, v openflow.FlowRemoved) error {
	r.log.Debug(fmt.Sprintf("FLOW_REMOVED is received: cookie=%v", v.Cookie()))

	if r.device == nil {
		r.log.Warning("Uninitialized device!")
		return nil
	}

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

func (r *Controller) findNeighborPort(deviceID string, portNum uint32) (*network.Port, error) {
	device := r.finder.Device(deviceID)
	if device == nil {
		return nil, fmt.Errorf("failed to find a neighbor device: id=%v", deviceID)
	}
	port := device.Port(portNum)
	if port == nil {
		return nil, fmt.Errorf("failed to find a neighbor port: deviceID=%v, portNum=%v", deviceID, portNum)
	}

	return port, nil
}

func (r *Controller) handleLLDP(inPort *network.Port, ethernet *protocol.Ethernet) error {
	lldp, err := getLLDP(ethernet.Payload)
	if err != nil {
		return err
	}
	deviceID, portNum, err := extractDeviceInfo(lldp)
	if err != nil {
		// Do nothing if this packet is not the one we sent
		r.log.Info("Ignoring a LLDP packet issued by an unknown device")
		return nil
	}
	port, err := r.findNeighborPort(deviceID, portNum)
	if err != nil {
		return err
	}
	r.watcher.DeviceLinked([2]*network.Port{inPort, port})

	return nil
}

func (r *Controller) addNewNode(inPort *network.Port, mac net.HardwareAddr) error {
	r.log.Debug(fmt.Sprintf("Adding a new node %v on %v..", mac, inPort.ID()))

	node := inPort.AddNode(mac)
	r.watcher.NodeAdded(node)

	return nil
}

func (r *Controller) isActivatedPort(p *network.Port) bool {
	// We assume that a port is in inactive state during 3 seconds after setting its value to avoid broadcast storm.
	return p.Duration().Seconds() > 3
}

func (r *Controller) OnPacketIn(f openflow.Factory, w trans.Writer, v openflow.PacketIn) error {
	r.log.Debug(fmt.Sprintf("PACKET_IN is received: inport=%v, reason=%v, tableID=%v, cookie=%v", v.InPort(), v.Reason(), v.TableID(), v.Cookie()))

	if r.device == nil {
		r.log.Warning("Uninitialized device!")
		return nil
	}

	ethernet, err := getEthernet(v.Data())
	if err != nil {
		return err
	}
	inPort := r.device.Port(v.InPort())
	if inPort == nil {
		return fmt.Errorf("failed to find a port: deviceID=%v, portNum=%v", r.device.ID(), v.InPort())
	}
	// Process LLDP, and then add an edge among two switches
	if isLLDP(ethernet) {
		return r.handleLLDP(inPort, ethernet)
	}
	// Do we know packet sender?
	if r.finder.Node(ethernet.SrcMAC) == nil {
		r.log.Debug(fmt.Sprintf("MAC learning... %v", ethernet.SrcMAC))

		// MAC learning
		if err := r.addNewNode(inPort, ethernet.SrcMAC); err != nil {
			return err
		}
	}
	// Do nothing if the ingress port is in inactive state
	if !r.isActivatedPort(inPort) {
		r.log.Info(fmt.Sprintf("Ignoring PACKET_IN from %v:%v because the ingress port is not in active state yet", r.device.ID(), v.InPort()))
		return nil
	}
	// Do nothing if the ingress port is an edge between switches and is disabled by STP.
	if r.finder.IsEdge(inPort) && !r.finder.IsEnabledBySTP(inPort) {
		r.log.Info(fmt.Sprintf("STP: ignoring PACKET_IN from %v:%v", r.device.ID(), v.InPort()))
		return nil
	}
	if err := r.handler.OnPacketIn(f, w, v); err != nil {
		return err
	}

	return r.processor.ProcessPacket(r.finder, ethernet, inPort)
}

// TODO: Use context to shutdown running controllers
func (r *Controller) Run() {
	if err := r.trans.Run(); err != nil && err != io.EOF {
		r.log.Err(fmt.Sprintf("OpenFlow transceiver abnormally terminated: %v", err))
	}
	r.trans.Close()
	if r.device != nil {
		r.log.Info(fmt.Sprintf("Session controller is disconnected (DPID=%v, AuxID=%v)", r.device.ID(), r.auxID))
		r.device.RemoveController(r.auxID)
		// Main connection?
		if r.auxID == 0 {
			// Assume the device is disconnected
			r.processor.ProcessDeviceClose(r.finder, r.device)
		}
	} else {
		r.log.Info(fmt.Sprintf("Session controller is disconnected (AuxID=%v)", r.auxID))
	}
}

func (r *Controller) Write(msg encoding.BinaryMarshaler) error {
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
	// Wildcard
	msg.SetTableID(0xFF)
	msg.SetFlowMatch(match)

	return w.Write(msg)
}
