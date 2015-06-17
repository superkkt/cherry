/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package of13

import (
	"encoding/binary"
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
)

type Config struct {
	err            error
	flags          uint16
	missSendLength uint16
}

func (r *Config) Flags() openflow.ConfigFlag {
	switch r.flags {
	case OFPC_FRAG_NORMAL:
		return openflow.FragNormal
	case OFPC_FRAG_DROP:
		return openflow.FragDrop
	case OFPC_FRAG_REASM:
		return openflow.FragReasm
	case OFPC_FRAG_MASK:
		return openflow.FragMask
	default:
		panic(fmt.Sprintf("unexpected config flag: %v", r.flags))
	}
}

func (r *Config) SetFlags(flags openflow.ConfigFlag) {
	switch flags {
	case openflow.FragNormal:
		r.flags = OFPC_FRAG_NORMAL
	case openflow.FragDrop:
		r.flags = OFPC_FRAG_DROP
	case openflow.FragReasm:
		r.flags = OFPC_FRAG_REASM
	case openflow.FragMask:
		r.flags = OFPC_FRAG_MASK
	default:
		r.err = fmt.Errorf("SetFlags: unexpected config flag: %v", flags)
	}
}

func (r *Config) MissSendLength() uint16 {
	return r.missSendLength
}

func (r *Config) SetMissSendLength(length uint16) {
	r.missSendLength = length
}

func (r *Config) Error() error {
	return r.err
}

type SetConfig struct {
	openflow.Message
	Config
}

func NewSetConfig(xid uint32) openflow.SetConfig {
	return &SetConfig{
		Message: openflow.NewMessage(openflow.OF13_VERSION, OFPT_SET_CONFIG, xid),
		Config: Config{
			flags:          OFPC_FRAG_NORMAL,
			missSendLength: 0xFFFF,
		},
	}
}

func (r *SetConfig) MarshalBinary() ([]byte, error) {
	if err := r.Error(); err != nil {
		return nil, err
	}

	v := make([]byte, 4)
	binary.BigEndian.PutUint16(v[0:2], r.flags)
	binary.BigEndian.PutUint16(v[2:4], r.missSendLength)
	r.SetPayload(v)

	return r.Message.MarshalBinary()
}

type GetConfigRequest struct {
	openflow.Message
}

func NewGetConfigRequest(xid uint32) openflow.GetConfigRequest {
	return &GetConfigRequest{
		Message: openflow.NewMessage(openflow.OF13_VERSION, OFPT_GET_CONFIG_REQUEST, xid),
	}
}

func (r *GetConfigRequest) MarshalBinary() ([]byte, error) {
	return r.Message.MarshalBinary()
}

type GetConfigReply struct {
	openflow.Message
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
	r.flags = binary.BigEndian.Uint16(payload[0:2])
	r.missSendLength = binary.BigEndian.Uint16(payload[2:4])

	return nil
}
