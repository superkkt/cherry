/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

type Hello struct {
	Message
}

func NewHello(version uint8, xid uint32) *Hello {
	return &Hello{
		// OFPT_HELLO
		Message: NewMessage(version, 0x0, xid),
	}
}

func (r *Hello) MarshalBinary() ([]byte, error) {
	return r.Message.MarshalBinary()
}

func (r *Hello) UnmarshalBinary(data []byte) error {
	return r.Message.UnmarshalBinary(data)
}
