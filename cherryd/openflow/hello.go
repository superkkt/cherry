/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

type Hello struct {
	header Header
}

func NewHello(version uint8, xid uint32) *Hello {
	return &Hello{
		header: Header{
			Version: version,
			Type:    0, // OFPT_HELLO
			XID:     xid,
		},
	}
}

func (r *Hello) Header() Header {
	return r.header
}

func (r *Hello) MarshalBinary() ([]byte, error) {
	return r.header.MarshalBinary()
}

func (r *Hello) UnmarshalBinary(data []byte) error {
	return r.header.UnmarshalBinary(data)
}
