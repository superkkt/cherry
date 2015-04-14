/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

import (
	"encoding/binary"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type Config struct {
	openflow.Message
	Flags       uint16
	MissSendLen uint16
}

type SetConfig struct {
	Config
}

func NewSetConfig(xid uint32, flags, missSendLen uint16) *SetConfig {
	return &SetConfig{
		Config{
			Message:     openflow.NewMessage(openflow.Ver13, OFPT_SET_CONFIG, xid),
			Flags:       flags,
			MissSendLen: missSendLen,
		},
	}
}

func (r *SetConfig) MarshalBinary() ([]byte, error) {
	v := make([]byte, 4)
	binary.BigEndian.PutUint16(v[0:2], r.Flags)
	binary.BigEndian.PutUint16(v[2:4], r.MissSendLen)
	r.SetPayload(v)

	return r.Message.MarshalBinary()
}

type GetConfigRequest struct {
	openflow.Message
}

func NewGetConfigRequest(xid uint32) *GetConfigRequest {
	return &GetConfigRequest{
		Message: openflow.NewMessage(openflow.Ver13, OFPT_GET_CONFIG_REQUEST, xid),
	}
}

func (r *GetConfigRequest) MarshalBinary() ([]byte, error) {
	return r.Message.MarshalBinary()
}

type GetConfigReply struct {
	Config
}

func (r *GetConfigReply) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}

	payload := r.Payload()
	if payload == nil || len(payload) < 4 {
		return openflow.ErrInvalidPacketLength
	}
	r.Flags = binary.BigEndian.Uint16(payload[0:2])
	r.MissSendLen = binary.BigEndian.Uint16(payload[2:4])

	return nil
}
