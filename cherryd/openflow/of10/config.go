/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of10

import (
	"encoding/binary"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type Config struct {
	header      openflow.Header
	Flags       uint16
	MissSendLen uint16
}

type SetConfig struct {
	Config
}

func NewSetConfig(xid uint32, flags, missSendLen uint16) *SetConfig {
	return &SetConfig{
		Config{
			header: openflow.Header{
				Version: openflow.Ver10,
				Type:    OFPT_SET_CONFIG,
				XID:     xid,
			},
			Flags:       flags,
			MissSendLen: missSendLen,
		},
	}
}

func (r *SetConfig) Header() openflow.Header {
	return r.header
}

func (r *SetConfig) MarshalBinary() ([]byte, error) {
	r.header.Length = 12
	header, err := r.header.MarshalBinary()
	if err != nil {
		return nil, err
	}

	v := make([]byte, r.header.Length)
	copy(v[0:8], header)
	binary.BigEndian.PutUint16(v[8:10], r.Flags)
	binary.BigEndian.PutUint16(v[10:12], r.MissSendLen)

	return v, nil
}

func (r *SetConfig) UnmarshalBinary(data []byte) error {
	return openflow.ErrUnsupportedUnmarshaling
}

type GetConfigRequest struct {
	header openflow.Header
}

func NewGetConfigRequest(xid uint32) *GetConfigRequest {
	return &GetConfigRequest{
		header: openflow.Header{
			Version: openflow.Ver10,
			Type:    OFPT_GET_CONFIG_REQUEST,
			XID:     xid,
		},
	}
}

func (r *GetConfigRequest) Header() openflow.Header {
	return r.header
}

func (r *GetConfigRequest) MarshalBinary() ([]byte, error) {
	r.header.Length = 8
	return r.header.MarshalBinary()
}

func (r *GetConfigRequest) UnmarshalBinary(data []byte) error {
	return openflow.ErrUnsupportedUnmarshaling
}

type GetConfigReply struct {
	Config
}

func (r *GetConfigReply) Header() openflow.Header {
	return r.header
}

func (r *GetConfigReply) MarshalBinary() ([]byte, error) {
	return nil, openflow.ErrUnsupportedMarshaling
}

func (r *GetConfigReply) UnmarshalBinary(data []byte) error {
	if err := r.header.UnmarshalBinary(data); err != nil {
		return err
	}
	if r.header.Length < 12 || len(data) < int(r.header.Length) {
		return openflow.ErrInvalidPacketLength
	}

	r.Flags = binary.BigEndian.Uint16(data[8:10])
	r.MissSendLen = binary.BigEndian.Uint16(data[10:12])

	return nil
}
