/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package device

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/net/protocol"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"golang.org/x/net/context"
	"net"
	"strconv"
	"sync/atomic"
	"time"
)

type FlowModConfig struct {
	Match                    openflow.Match
	Action                   openflow.Action
	IdleTimeout, HardTimeout uint16
	Priority                 uint16
}

type Transceiver interface {
	Run(context.Context)
	newMatch() openflow.Match
	newAction() openflow.Action
	sendBarrierRequest() error
	addFlowMod(conf FlowModConfig) error
	packetOut(inport openflow.InPort, action openflow.Action, data []byte) error
	flood(inPort openflow.InPort, data []byte) error
}

type baseTransceiver struct {
	stream       *openflow.Stream
	log          Logger
	xid          uint32
	device       *Device
	version      uint8
	lldpExplored atomic.Value
	processor    PacketProcessor
}

type PacketProcessor interface {
	Run(eth *protocol.Ethernet, ingress Point) error
}

func (r *baseTransceiver) getTransactionID() uint32 {
	// xid will be started from 1, not 0.
	return atomic.AddUint32(&r.xid, 1)
}

func (r *baseTransceiver) handleEchoRequest(msg *openflow.EchoRequest) error {
	reply := openflow.NewEchoReply(msg.Version(), msg.TransactionID(), msg.Data)
	if err := openflow.WriteMessage(r.stream, reply); err != nil {
		return fmt.Errorf("failed to send echo reply_message: %v", err)
	}

	// XXX: debugging
	r.log.Printf("EchoRequest: %+v", msg)

	return nil
}

func (r *baseTransceiver) handleEchoReply(msg *openflow.EchoReply) error {
	if msg.Data == nil || len(msg.Data) != 8 {
		return errors.New("Invalid echo reply data")
	}
	timestamp := int64(binary.BigEndian.Uint64(msg.Data))
	latency := time.Now().UnixNano() - timestamp

	// XXX: debugging
	r.log.Printf("EchoReply: latency=%vms", latency/1000/1000)

	return nil
}

func (r *baseTransceiver) sendEchoRequest(version uint8) error {
	data := make([]byte, 8)
	timestamp := time.Now().UnixNano()
	binary.BigEndian.PutUint64(data, uint64(timestamp))

	echo := openflow.NewEchoRequest(version, r.getTransactionID(), data)
	if err := openflow.WriteMessage(r.stream, echo); err != nil {
		return fmt.Errorf("failed to send echo_request message: %v", err)
	}

	return nil
}

func (r *baseTransceiver) newLLDPEtherFrame(port openflow.Port) ([]byte, error) {
	if r.device == nil {
		return nil, errors.New("newLLDPEtherFrame: nil device")
	}

	lldp := &protocol.LLDP{
		ChassisID: protocol.LLDPChassisID{
			SubType: 7, // Locally assigned alpha-numeric string
			Data:    []byte(fmt.Sprintf("%v", r.device.DPID)),
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

// TODO: Implement to close the connection if we miss several echo replies
func (r *baseTransceiver) pinger(ctx context.Context, version uint8) {
	ticker := time.NewTicker(time.Duration(15) * time.Second)
	for {
		select {
		case <-ticker.C:
			if err := r.sendEchoRequest(version); err != nil {
				r.log.Printf("failed to send echo request: %v", err)
				return
			}
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (r *baseTransceiver) isLLDPExplored() bool {
	return r.lldpExplored.Load().(bool)
}

func (r *baseTransceiver) startLLDPTimer() {
	// We will serve PACKET_IN after n seconds to make sure LLDPs explore whole network topology.
	go func() {
		time.Sleep(2 * time.Second)
		r.lldpExplored.Store(true)
	}()
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

func (r *baseTransceiver) handleLLDP(inPort uint32, eth *protocol.Ethernet) error {
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
	port, ok := r.device.Port(uint(inPort))
	if !ok {
		// XXX: debugging
		r.log.Print("NIL port..\n")
		return nil
	}

	p1 := &Point{r.device, inPort}
	p2 := &Point{neighbor, uint32(portNum)}
	edge := newEdge(p1, p2, calculateEdgeWeight(port.Speed()))
	Switches.graph.AddEdge(edge)

	// XXX: debugging
	fmt.Printf("LLDP from %v:%v, Edge ID=%v\n", dpid, portNum, edge.ID())

	return nil
}

func (r *baseTransceiver) sendLLDP(port openflow.Port) error {
	lldp, err := r.newLLDPEtherFrame(port)
	if err != nil {
		return err
	}
	action := r.device.NewAction()
	action.SetOutput(port.Number())
	inport := openflow.NewInPort()
	if err := r.device.PacketOut(inport, action, lldp); err != nil {
		return err
	}

	return nil
}

func (r *baseTransceiver) updatePortStatus(port openflow.Port) error {
	if port.IsPortDown() || port.IsLinkDown() {
		p := Point{r.device, uint32(port.Number())}
		// Recalculate minimum spanning tree
		Switches.graph.RemoveEdge(p)
		// Remove learned MAC address
		Hosts.remove(p)
	} else {
		// Update device graph by sending an LLDP packet
		if err := r.sendLLDP(port); err != nil {
			return err
		}
	}
	// Update port status
	r.device.setPort(port.Number(), port)

	// XXX: debugging
	{
		if port.IsPortDown() {
			fmt.Printf("PortDown: %v/%v\n", r.device.DPID, port.Number())
		}
		if port.IsLinkDown() {
			fmt.Printf("LinkDown: %v/%v\n", r.device.DPID, port.Number())
		}
	}

	return nil
}

func (r *baseTransceiver) handleIncoming(inPort uint32, packet []byte) error {
	eth := new(protocol.Ethernet)
	if err := eth.UnmarshalBinary(packet); err != nil {
		return err
	}
	// XXX: debugging
	fmt.Printf("Ethernet: %+v", eth)

	// LLDP?
	if eth.Type == 0x88CC {
		if err := r.handleLLDP(inPort, eth); err != nil {
			return err
		}
		return nil
	}

	p := Point{r.device, inPort}
	// MAC learning if we are not on an edge between switches
	if !Switches.graph.IsEdge(p) {
		Hosts.add(eth.SrcMAC, p)
	}
	// Do nothing if LLDPs we sent are still exploring network topology.
	if !r.isLLDPExplored() {
		// XXX: debugging
		r.log.Printf("Ignoring PACKET_IN on %v/%v due to LLDP exploring...\n", p.Node.DPID, p.Port)
		return nil
	}
	// Do nothing if the ingress port is an edge between switches and is disabled by STP.
	if Switches.graph.IsEdge(p) && !Switches.graph.IsEnabledPoint(p) {
		// XXX: debugging
		r.log.Printf("Ignoring PACKET_IN on %v/%v due to disabled port by STP...\n", p.Node.DPID, p.Port)
		return nil
	}

	if r.processor == nil {
		return nil
	}
	return r.processor.Run(eth, p)
}

func NewTransceiver(conn net.Conn, log Logger, p PacketProcessor) (Transceiver, error) {
	stream := openflow.NewStream(conn)
	stream.SetReadTimeout(5 * time.Second)
	msg, err := openflow.ReadMessage(stream)
	if err != nil {
		return nil, err
	}

	// The first message should be HELLO
	if msg.Type() != 0x0 {
		return nil, errors.New("negotiation error: missing HELLO message")
	}

	if msg.Version() < openflow.Ver13 {
		return NewOF10Transceiver(stream, log, p), nil
	} else {
		return NewOF13Transceiver(stream, log, p), nil
	}
}
