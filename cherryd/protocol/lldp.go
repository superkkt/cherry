/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package protocol

import (
	"encoding/binary"
	"errors"
)

type LLDPChassisID struct {
	SubType uint8
	Data    []byte
}

type LLDPPortID struct {
	SubType uint8
	Data    []byte
}

type LLDP struct {
	ChassisID LLDPChassisID
	PortID    LLDPPortID
	TTL       uint16
}

func (r *LLDP) marshalChassisID() ([]byte, error) {
	if r.ChassisID.Data == nil {
		return nil, errors.New("nil chassis ID")
	}
	if len(r.ChassisID.Data) > 255 {
		return nil, errors.New("too long chassis ID")
	}

	var header uint16
	length := len(r.ChassisID.Data) + 1
	header = uint16(1<<9 | (length & 0x1FF))

	v := make([]byte, length+2)
	binary.BigEndian.PutUint16(v[0:2], header)
	v[2] = r.ChassisID.SubType
	copy(v[3:], r.ChassisID.Data)

	return v, nil
}

func (r *LLDP) marshalPortID() ([]byte, error) {
	if r.PortID.Data == nil {
		return nil, errors.New("nil port ID")
	}
	if len(r.PortID.Data) > 255 {
		return nil, errors.New("too long port ID")
	}

	var header uint16
	length := len(r.PortID.Data) + 1
	header = uint16(2<<9 | (length & 0x1FF))

	v := make([]byte, length+2)
	binary.BigEndian.PutUint16(v[0:2], header)
	v[2] = r.PortID.SubType
	copy(v[3:], r.PortID.Data)

	return v, nil
}

func (r *LLDP) marshalTTL() ([]byte, error) {
	var header uint16
	length := 2
	header = uint16(3<<9 | (length & 0x1FF))

	v := make([]byte, 4)
	binary.BigEndian.PutUint16(v[0:2], header)
	binary.BigEndian.PutUint16(v[2:4], r.TTL)

	return v, nil
}

func (r *LLDP) MarshalBinary() ([]byte, error) {
	v := make([]byte, 0)

	chassis, err := r.marshalChassisID()
	if err != nil {
		return nil, err
	}
	v = append(v, chassis...)

	port, err := r.marshalPortID()
	if err != nil {
		return nil, err
	}
	v = append(v, port...)

	ttl, err := r.marshalTTL()
	if err != nil {
		return nil, err
	}
	v = append(v, ttl...)

	// End of TLV
	v = append(v, []byte{0, 0}...)

	return v, nil
}

func (r *LLDP) unmarshalChassisID(data []byte) (n int, err error) {
	length := len(data)
	if length < 2 {
		return 0, errors.New("invalid chassis ID TLV length")
	}

	header := binary.BigEndian.Uint16(data[0:2])
	tlvType := (header >> 9) & 0x7F
	if tlvType != 1 {
		return 0, errors.New("invalid chassis ID TLV type")
	}
	tlvLength := header & 0x1FF
	if length < int(tlvLength+2) {
		return 0, errors.New("invalid chassis ID TLV length")
	}
	r.ChassisID = LLDPChassisID{
		SubType: data[2],
		Data:    data[3 : 3+tlvLength-1],
	}

	return int(tlvLength + 2), nil
}

func (r *LLDP) unmarshalPortID(data []byte) (n int, err error) {
	length := len(data)
	if length < 2 {
		return 0, errors.New("invalid port ID TLV length")
	}

	header := binary.BigEndian.Uint16(data[0:2])
	tlvType := (header >> 9) & 0x7F
	if tlvType != 2 {
		return 0, errors.New("invalid port ID TLV type")
	}
	tlvLength := header & 0x1FF
	if length < int(tlvLength+2) {
		return 0, errors.New("invalid port ID TLV length")
	}
	r.PortID = LLDPPortID{
		SubType: data[2],
		Data:    data[3 : 3+tlvLength-1],
	}

	return int(tlvLength + 2), nil
}

func (r *LLDP) unmarshalTTL(data []byte) (n int, err error) {
	length := len(data)
	if length < 2 {
		return 0, errors.New("invalid TTL TLV length")
	}

	header := binary.BigEndian.Uint16(data[0:2])
	tlvType := (header >> 9) & 0x7F
	if tlvType != 3 {
		return 0, errors.New("invalid TTL TLV type")
	}
	tlvLength := header & 0x1FF
	if length < int(tlvLength+2) {
		return 0, errors.New("invalid TTL TLV length")
	}
	r.TTL = binary.BigEndian.Uint16(data[2:4])

	return int(tlvLength + 2), nil
}

func (r *LLDP) UnmarshalBinary(data []byte) error {
	offset := 0
	length := len(data)

	// From IEEE 802.1AB-2009:
	//
	// a) Three mandatory TLVs shall be included at the beginning of each LLDPDU and shall be in the order shown.
	// 	1) Chassis ID TLV
	// 	2) Port ID TLV
	// 	3) Time To Live TLV
	// b) Optional TLVs as selected by network management (may be inserted in any order).
	n, err := r.unmarshalChassisID(data)
	if err != nil {
		return err
	}
	offset += n

	if length < offset {
		return errors.New("invalid LLDP packet length")
	}
	n, err = r.unmarshalPortID(data[offset:])
	if err != nil {
		return err
	}
	offset += n

	if length < offset {
		return errors.New("invalid LLDP packet length")
	}
	_, err = r.unmarshalTTL(data[offset:])
	if err != nil {
		return err
	}

	return nil
}
