/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding/binary"
	"net"
	"strings"
)

type FeaturesReplyMessage struct {
	Header
	DPID         uint64
	NumBuffers   uint32
	NumTables    uint8
	Capabilities uint32
	Actions      uint32
	Ports        []PhysicalPort
}

func unmarshalPhysicalPort(data []byte) (PhysicalPort, error) {
	if len(data) < 48 {
		return PhysicalPort{}, ErrInvalidPacketLength
	}

	p := PhysicalPort{}
	p.Number = binary.BigEndian.Uint16(data[0:2])
	p.MAC = make(net.HardwareAddr, 6)
	copy(p.MAC, data[2:8])
	p.Name = strings.TrimRight(string(data[8:24]), "\x00")
	p.config = binary.BigEndian.Uint32(data[24:28])
	p.state = binary.BigEndian.Uint32(data[28:32])
	p.current = binary.BigEndian.Uint32(data[32:36])
	p.advertised = binary.BigEndian.Uint32(data[36:40])
	p.supported = binary.BigEndian.Uint32(data[40:44])
	p.peer = binary.BigEndian.Uint32(data[44:48])

	return p, nil
}

func (r *FeaturesReplyMessage) UnmarshalBinary(data []byte) error {
	header := &Header{}
	if err := header.UnmarshalBinary(data); err != nil {
		return err
	}
	if len(data) != int(header.Length) {
		return ErrInvalidPacketLength
	}

	r.Header = *header
	r.DPID = binary.BigEndian.Uint64(data[8:16])
	r.NumBuffers = binary.BigEndian.Uint32(data[16:20])
	r.NumTables = data[20]
	r.Capabilities = binary.BigEndian.Uint32(data[24:28])
	r.Actions = binary.BigEndian.Uint32(data[28:32])

	nPorts := (header.Length - 32) / 48
	if nPorts == 0 {
		return nil
	}
	r.Ports = make([]PhysicalPort, nPorts)
	for i := uint16(0); i < nPorts; i++ {
		buf := data[32+i*48:]
		port, err := unmarshalPhysicalPort(buf)
		if err != nil {
			return err
		}
		r.Ports[i] = port
	}

	return nil
}
