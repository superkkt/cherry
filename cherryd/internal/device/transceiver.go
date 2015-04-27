/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package device

import (
	"encoding/binary"
	"errors"
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/net/protocol"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"golang.org/x/net/context"
	"net"
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

func (r *baseTransceiver) IsLLDPExplored() bool {
	return r.lldpExplored.Load().(bool)
}

func (r *baseTransceiver) LLDPTimer() {
	// We will serve PACKET_IN after n seconds to make sure LLDPs explore whole network topology.
	go func() {
		time.Sleep(2 * time.Second)
		r.lldpExplored.Store(true)
	}()
}

func NewTransceiver(conn net.Conn, log Logger) (Transceiver, error) {
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
		return NewOF10Transceiver(stream, log), nil
	} else {
		return NewOF13Transceiver(stream, log), nil
	}
}
