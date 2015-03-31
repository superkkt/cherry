/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"encoding/binary"
)

type ConfigMessage struct {
	Header
	Flags       ConfigFlag
	MissSendLen uint16
}

func (r *ConfigMessage) MarshalBinary() ([]byte, error) {
	r.Header.Length = 12
	header, err := r.Header.MarshalBinary()
	if err != nil {
		return nil, err
	}

	v := make([]byte, r.Header.Length)
	copy(v[0:8], header)
	binary.BigEndian.PutUint16(v[8:10], uint16(r.Flags))
	binary.BigEndian.PutUint16(v[10:12], r.MissSendLen)

	return v, nil
}

func (r *ConfigMessage) UnmarshalBinary(data []byte) error {
	header := &Header{}
	if err := header.UnmarshalBinary(data); err != nil {
		return err
	}
	if len(data) != int(header.Length) {
		return ErrInvalidPacketLength
	}

	r.Header = *header
	r.Flags = ConfigFlag(binary.BigEndian.Uint16(data[8:10]))
	r.MissSendLen = binary.BigEndian.Uint16(data[10:12])

	return nil
}

type SetConfigMessage struct {
	ConfigMessage
}

func (r *SetConfigMessage) MarshalBinary() ([]byte, error) {
	return r.ConfigMessage.MarshalBinary()
}

type GetConfigRequestMessage struct {
	Header
	// This message does not contain a body beyond the header
}

func (r *GetConfigRequestMessage) MarshalBinary() ([]byte, error) {
	r.Header.Length = 8
	return r.Header.MarshalBinary()
}

type GetConfigReplyMessage struct {
	ConfigMessage
}

func (r *GetConfigReplyMessage) UnmarshalBinary(data []byte) error {
	c := ConfigMessage{}
	if err := c.UnmarshalBinary(data); err != nil {
		return err
	}
	r.ConfigMessage = c

	return nil
}
