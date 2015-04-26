/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package device

import (
	"bytes"
	"errors"
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/net/protocol"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"git.sds.co.kr/cherry.git/cherryd/openflow/of13"
	"golang.org/x/net/context"
	"strconv"
	"time"
)

type OF13Transceiver struct {
	baseTransceiver
	auxID uint8
}

func NewOF13Transceiver(stream *openflow.Stream, log Logger) *OF13Transceiver {
	return &OF13Transceiver{
		baseTransceiver: baseTransceiver{
			stream:  stream,
			log:     log,
			version: openflow.Ver13,
		},
	}
}

func (r *OF13Transceiver) sendHello() error {
	hello := openflow.NewHello(r.version, r.getTransactionID())
	return openflow.WriteMessage(r.stream, hello)
}

func (r *OF13Transceiver) sendFeaturesRequest() error {
	feature := of13.NewFeaturesRequest(r.getTransactionID())
	return openflow.WriteMessage(r.stream, feature)
}

func (r *OF13Transceiver) sendBarrierRequest() error {
	barrier := of13.NewBarrierRequest(r.getTransactionID())
	return openflow.WriteMessage(r.stream, barrier)
}

func (r *OF13Transceiver) sendSetConfig(flags, missSendLen uint16) error {
	msg := of13.NewSetConfig(r.getTransactionID(), flags, missSendLen)
	return openflow.WriteMessage(r.stream, msg)
}

func (r *OF13Transceiver) sendDescriptionRequest() error {
	msg := of13.NewDescriptionRequest(r.getTransactionID())
	return openflow.WriteMessage(r.stream, msg)
}

func (r *OF13Transceiver) sendPortDescriptionRequest() error {
	msg := of13.NewPortDescriptionRequest(r.getTransactionID())
	return openflow.WriteMessage(r.stream, msg)
}

func (r *OF13Transceiver) addFlowMod(conf FlowModConfig) error {
	c := &of13.FlowModConfig{
		// TODO: set Cookie
		// TODO: set TableID depend on their usage using flow removed message
		TableID:     0,
		IdleTimeout: conf.IdleTimeout,
		HardTimeout: conf.HardTimeout,
		Priority:    conf.Priority,
		Match:       conf.Match,
		Instruction: &of13.ApplyAction{Action: conf.Action},
	}
	msg := of13.NewFlowModAdd(r.getTransactionID(), c)
	return openflow.WriteMessage(r.stream, msg)
}

func (r *OF13Transceiver) newMatch() openflow.Match {
	return of13.NewMatch()
}

func (r *OF13Transceiver) newAction() openflow.Action {
	return of13.NewAction()
}

func (r *OF13Transceiver) packetOut(inport openflow.InPort, action openflow.Action, data []byte) error {
	msg := of13.NewPacketOut(r.getTransactionID(), inport, action, data)
	return openflow.WriteMessage(r.stream, msg)
}

func (r *OF13Transceiver) flood(inPort openflow.InPort, data []byte) error {
	if r.device == nil {
		panic("OF13Transceiver: flood on nil device")
	}

	action := of13.NewAction()
	action.SetOutput(of13.OFPP_ALL)
	msg := of13.NewPacketOut(r.getTransactionID(), inPort, action, data)
	return openflow.WriteMessage(r.stream, msg)
}

func (r *OF13Transceiver) removeAllFlows() error {
	c := &of13.FlowModConfig{
		TableID: of13.OFPTT_ALL,
		// All wildcarded match
		Match: of13.NewMatch(),
	}

	return openflow.WriteMessage(r.stream, of13.NewFlowModDelete(r.getTransactionID(), c))
}

func (r *OF13Transceiver) setTableMiss() error {
	// FIXME: Is it okay to set a table-miss entry for the first table only? ONOS also does same thing..
	packetin := of13.NewAction()
	packetin.SetOutput(of13.OFPP_CONTROLLER)

	c := &of13.FlowModConfig{
		TableID: 0,
		// Permanent flow entry
		IdleTimeout: 0, HardTimeout: 0,
		// Table-miss entry should have zero priority
		Priority: 0,
		// All wildcarded match
		Match:       of13.NewMatch(),
		Instruction: &of13.ApplyAction{Action: packetin},
	}

	return openflow.WriteMessage(r.stream, of13.NewFlowModAdd(r.getTransactionID(), c))
}

func (r *OF13Transceiver) handleFeaturesReply(msg *of13.FeaturesReply) error {
	v := Switches.Get(msg.DPID)
	if v == nil {
		v = newDevice(msg.DPID)
	}
	Switches.add(msg.DPID, v)

	r.device = v
	r.device.NumBuffers = uint(msg.NumBuffers)
	r.device.NumTables = uint(msg.NumTables)
	r.device.addTransceiver(uint(msg.AuxID), r)
	r.auxID = msg.AuxID

	// XXX: debugging
	//	{
	//		r.log.Printf("FeaturesReply: %+v", msg)
	//		getconfig := of13.NewGetConfigRequest(r.getTransactionID())
	//		if err := openflow.WriteMessage(r.stream, getconfig); err != nil {
	//			return err
	//		}
	//
	//		match := r.newMatch()
	//		match.SetInPort(1)
	//		action := r.newAction()
	//		action.SetOutput(openflow.PortTable)
	//		action.SetSrcMAC(openflow.ZeroMAC)
	//		conf := FlowModConfig{
	//			IdleTimeout: 20,
	//			Priority:    10,
	//			Match:       match,
	//			Action:      action,
	//		}
	//		if err := r.addFlowMod(conf); err != nil {
	//			return err
	//		}
	//
	//		lldp := []byte{0x01, 0x80, 0xc2, 0x00, 0x00, 0x0e, 0x00, 0x01, 0xe8, 0xd8, 0x0f, 0x32, 0x88, 0xcc, 0x02, 0x07, 0x04, 0x00, 0x01, 0xe8, 0xd8, 0x0f, 0x25, 0x04, 0x06, 0x05, 0x73, 0x77, 0x70, 0x31, 0x33, 0x06, 0x02, 0x00, 0x78, 0x0a, 0x07, 0x63, 0x75, 0x6d, 0x75, 0x6c, 0x75, 0x73, 0x0c, 0x34, 0x43, 0x75, 0x6d, 0x75, 0x6c, 0x75, 0x73, 0x20, 0x4c, 0x69, 0x6e, 0x75, 0x78, 0x20, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x20, 0x32, 0x2e, 0x35, 0x2e, 0x31, 0x20, 0x72, 0x75, 0x6e, 0x6e, 0x69, 0x6e, 0x67, 0x20, 0x6f, 0x6e, 0x20, 0x64, 0x6e, 0x69, 0x20, 0x65, 0x74, 0x2d, 0x37, 0x34, 0x34, 0x38, 0x62, 0x66, 0x0e, 0x04, 0x00, 0x14, 0x00, 0x14, 0x08, 0x05, 0x73, 0x77, 0x70, 0x31, 0x33, 0x00, 0x00}
	//		if err := r.packetOut(openflow.NewInPort(), action, lldp); err != nil {
	//			return err
	//		}
	//
	//		table := of13.NewTableFeaturesRequest(r.getTransactionID())
	//		if err := openflow.WriteMessage(r.stream, table); err != nil {
	//			return err
	//		}
	//
	//		stats := of13.NewFlowStatsRequest(r.getTransactionID(), of13.OFPTT_ALL, 0, 0, of13.NewMatch())
	//		if err := openflow.WriteMessage(r.stream, stats); err != nil {
	//			return err
	//		}
	//	}

	return nil
}

func (r *OF13Transceiver) handleGetConfigReply(msg *of13.GetConfigReply) error {
	// XXX: debugging
	{
		r.log.Printf("GetConfigReply: %+v", msg)
	}

	return nil
}

func (r *OF13Transceiver) handleDescriptionReply(msg *of13.DescriptionReply) error {
	r.device.Manufacturer = msg.Manufacturer
	r.device.Hardware = msg.Hardware
	r.device.Software = msg.Software
	r.device.Serial = msg.Serial
	r.device.Description = msg.Description

	// XXX: debugging
	{
		r.log.Printf("DescriptionReply: %+v", msg)
	}

	return nil
}

func (r *OF13Transceiver) sendLLDP(port openflow.Port) error {
	lldp, err := r.newLLDPEtherFrame(port)
	if err != nil {
		return err
	}
	action := of13.NewAction()
	action.SetOutput(port.Number())
	inport := openflow.NewInPort()
	inport.SetPort(of13.OFPP_ANY)
	if err := r.packetOut(inport, action, lldp); err != nil {
		return err
	}

	return nil
}

func (r *OF13Transceiver) handlePortDescriptionReply(msg *of13.PortDescriptionReply) error {
	if r.device == nil {
		r.log.Printf("we got port_description_reply before description_reply!")
		time.Sleep(5 * time.Second)
		// Resend port description request
		return r.sendPortDescriptionRequest()
	}
	for _, v := range msg.Ports {
		// Reserved port?
		if v.Number() > of13.OFPP_MAX {
			continue
		}
		// Add new port information
		r.device.setPort(uint(v.Number()), v)
		if err := r.sendLLDP(v); err != nil {
			return err
		}
		// XXX: debugging
		r.log.Printf("Port: %+v", v)
	}

	// XXX: debugging
	{
		r.log.Printf("PortDescriptionReply: %+v", msg)
	}

	return nil
}

func (r *OF13Transceiver) handleError(msg *openflow.Error) error {
	r.log.Printf("Error: version=%v, xid=%v, type=%v, code=%v", msg.Version(), msg.TransactionID(), msg.Class, msg.Code)
	return nil
}

func (r *OF13Transceiver) handlePortStatus(msg *of13.PortStatus) error {
	if r.device == nil {
		r.log.Print("PortStatus is received, but we don't have a switch device yet!")
		return nil
	}

	// Port removed?
	switch msg.Reason {
	case of13.OFPPR_DELETE:
		Switches.graph.RemoveEdge(Point{r.device, uint32(msg.Port.Number())})
		// TODO: Remove learned MAC address in hosts DB
	case of13.OFPPR_ADD:
		// Update device graph by sending an LLDP packet
		if err := r.sendLLDP(msg.Port); err != nil {
			return err
		}
	case of13.OFPPR_MODIFY:
		// FIXME: Should we do something in here?
	}
	// Update port status
	r.device.setPort(msg.Port.Number(), msg.Port)

	// XXX: debugging
	{
		r.log.Printf("PortStatus: %+v, Port: %+v", msg, *msg.Port)
	}

	return nil
}

func (r *OF13Transceiver) handleFlowRemoved(msg *of13.FlowRemoved) error {
	// XXX: debugging
	{
		r.log.Printf("FlowRemoved: %+v, match=%+v", msg, msg.Match)
	}

	return nil
}

func isOurLLDP(p *protocol.LLDP) bool {
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

func getDeviceInfo(p *protocol.LLDP) (dpid uint64, port uint32, err error) {
	dpid, err = strconv.ParseUint(string(p.ChassisID.Data), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	// PortID.Data string consists of "cherry/" and port number
	portID, err := strconv.ParseUint(string(p.PortID.Data[7:]), 10, 32)
	if err != nil {
		return 0, 0, err
	}

	return dpid, uint32(portID), nil
}

func (r *OF13Transceiver) handleLLDP(msg *of13.PacketIn, eth *protocol.Ethernet) error {
	// XXX: debugging
	r.log.Printf("LLDP is received: %+v", eth)

	if r.device == nil {
		return errors.New("handleLLDP: nil device")
	}

	lldp := new(protocol.LLDP)
	if err := lldp.UnmarshalBinary(eth.Payload); err != nil {
		return err
	}
	// XXX: debugging
	r.log.Printf("LLDP: %+v\n", lldp)

	if isOurLLDP(lldp) == false {
		// Do nothing if this packet is not the one we sent
		return nil
	}
	dpid, portNum, err := getDeviceInfo(lldp)
	if err != nil {
		// Do nothing if this packet is not the one we sent
		return nil
	}

	neighbor := Switches.Get(dpid)
	if neighbor == nil {
		// XXX: debugging
		r.log.Print("NIL neighbor..\n")
		return nil
	}
	port, ok := r.device.Port(uint(msg.InPort))
	if !ok {
		// XXX: debugging
		r.log.Print("NIL port..\n")
		return nil
	}

	p1 := &Point{r.device, msg.InPort}
	p2 := &Point{neighbor, uint32(portNum)}
	edge := newEdge(p1, p2, calculateEdgeWeight(port.Speed()))
	Switches.graph.AddEdge(edge)

	// XXX: debugging
	fmt.Printf("LLDP from %v:%v, Edge ID=%v\n", dpid, portNum, edge.ID())

	return nil
}

func (r *OF13Transceiver) handlePacketIn(msg *of13.PacketIn) error {
	eth := new(protocol.Ethernet)
	if err := eth.UnmarshalBinary(msg.Data); err != nil {
		return err
	}

	// LLDP?
	if eth.Type == 0x88CC {
		if err := r.handleLLDP(msg, eth); err != nil {
			return err
		}
		return nil
	}

	// Do nothing if the ingress port is an edge between switches and is disabled by STP.
	p := Point{r.device, msg.InPort}
	if Switches.graph.IsEdge(p) && !Switches.graph.IsEnabledPoint(p) {
		// XXX: debugging
		r.log.Printf("Ignoring PACKET_IN on %v/%v...\n", p.Node.DPID, p.Port)
		return nil
	}

	// XXX: debugging
	{
		r.log.Printf("PacketIn: %+v", msg)
	}

	// TODO: Send this one to plugins

	// XXX: L2 MAC learning switch
	{
		flood := func(port uint32, data []byte) error {
			// XXX: debugging
			r.log.Print("Flooding..")

			// Flooding
			inPort := openflow.NewInPort()
			inPort.SetPort(uint(port))
			return r.flood(inPort, data)
		}

		r.log.Printf("Ethernet: %+v", eth)

		return flood(msg.InPort, msg.Data)
		/*
			// ARP?
			if eth.Type == 0x0806 {
				// XXX: debugging
				r.log.Print("ARP is received..\n")
				return flood(msg.InPort, msg.Data)
			}

			conn, ok := Hosts.Find(eth.DstMAC)
			if !ok {
				// XXX: debugging
				r.log.Printf("Failed to find the destination MAC: %v\n", eth.DstMAC)
				return flood(msg.InPort, msg.Data)
			}
			path := Switches.graph.FindPath(r.device, conn.Device)
			// Empty path means the destination is not connected with this device that sent PACKET_IN.
			if len(path) == 0 {
				// XXX: debugging
				r.log.Printf("We don't know the path to the destintion: %v\n", eth.DstMAC)
				// FIXME: Flood? or Drop?
				return flood(msg.InPort, msg.Data)
			}

			// XXX: debugging
			r.log.Printf("Sending PACKET_OUT to %v..\n", eth.DstMAC)
			action := conn.Device.NewAction()
			action.SetOutput(uint(conn.Port))
			return conn.Device.PacketOut(openflow.NewInPort(), action, msg.Data)
		*/
	}

	return nil
}

func (r *OF13Transceiver) handleMessage(msg openflow.Incoming) error {
	if msg.Version() != r.version {
		return errors.New("unexpected openflow protocol version!")
	}

	switch v := msg.(type) {
	case *openflow.EchoRequest:
		return r.handleEchoRequest(v)
	case *openflow.EchoReply:
		return r.handleEchoReply(v)
	case *openflow.Error:
		return r.handleError(v)
	case *of13.FeaturesReply:
		return r.handleFeaturesReply(v)
	case *of13.GetConfigReply:
		return r.handleGetConfigReply(v)
	case *of13.DescriptionReply:
		return r.handleDescriptionReply(v)
	case *of13.PortDescriptionReply:
		return r.handlePortDescriptionReply(v)
	case *of13.PortStatus:
		return r.handlePortStatus(v)
	case *of13.FlowRemoved:
		return r.handleFlowRemoved(v)
	case *of13.PacketIn:
		return r.handlePacketIn(v)
	default:
		r.log.Printf("Unsupported message type: version=%v, type=%v", msg.Version(), msg.Type())
		return nil
	}

	return nil
}

func (r *OF13Transceiver) cleanup() {
	if r.device == nil {
		return
	}

	if r.device.removeTransceiver(uint(r.auxID)) == 0 {
		Switches.remove(r.device.DPID)
	}
}

func (r *OF13Transceiver) init() error {
	if err := r.sendHello(); err != nil {
		return fmt.Errorf("failed to send hello message: %v", err)
	}
	if err := r.sendSetConfig(of13.OFPC_FRAG_NORMAL, 0xFFFF); err != nil {
		return fmt.Errorf("failed to send set_config message: %v", err)
	}
	if err := r.sendFeaturesRequest(); err != nil {
		return fmt.Errorf("failed to send features_request message: %v", err)
	}
	if err := r.sendDescriptionRequest(); err != nil {
		return fmt.Errorf("failed to send description_request message: %v", err)
	}
	if err := r.setTableMiss(); err != nil {
		return fmt.Errorf("failed to set table_miss flow entry: %v", err)
	}
	// Make sure that description_reply is received before port_description_reply
	if err := r.sendBarrierRequest(); err != nil {
		return fmt.Errorf("failed to send barrier_request: %v", err)
	}
	if err := r.sendPortDescriptionRequest(); err != nil {
		return fmt.Errorf("failed to send port_description_request message: %v", err)
	}

	return nil
}

func (r *OF13Transceiver) Run(ctx context.Context) {
	defer r.cleanup()

	r.stream.SetReadTimeout(1 * time.Second)
	r.stream.SetWriteTimeout(5 * time.Second)
	if err := r.init(); err != nil {
		r.log.Printf("init: %v", err)
		return
	}
	go r.pinger(ctx, r.version)

	// Reader goroutine
	receivedMsg := make(chan openflow.Incoming)
	go func() {
		for {
			msg, err := openflow.ReadMessage(r.stream)
			if err != nil {
				switch {
				case openflow.IsTimeout(err):
					// Ignore timeout error
					continue
				case err == openflow.ErrUnsupportedMessage:
					r.log.Print(err)
					continue
				default:
					r.log.Print(err)
					close(receivedMsg)
					return
				}
			}
			receivedMsg <- msg
		}
	}()

	// Infinite loop
	for {
		select {
		case msg, ok := <-receivedMsg:
			if !ok {
				return
			}
			if err := r.handleMessage(msg); err != nil {
				r.log.Print(err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
