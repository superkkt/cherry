/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

type Echo struct {
	header Header
	Data   []byte
}

func (r *Echo) Header() Header {
	return r.header
}

func (r *Echo) MarshalBinary() ([]byte, error) {
	v := make([]byte, r.header.Length)

	header, err := r.header.MarshalBinary()
	if err != nil {
		return nil, err
	}
	copy(v[0:8], header)

	if r.Data != nil && len(r.Data) > 0 {
		copy(v[8:], r.Data)
	}

	return v, nil
}

func (r *Echo) UnmarshalBinary(data []byte) error {
	if data == nil || len(data) == 0 {
		return ErrInvalidPacketLength
	}

	if err := r.header.UnmarshalBinary(data); err != nil {
		return err
	}
	if len(data) < int(r.header.Length) {
		return ErrInvalidPacketLength
	}

	if r.header.Length > 8 {
		r.Data = make([]byte, r.header.Length-8)
		copy(r.Data, data[8:])
	}

	return nil
}

type EchoRequest struct {
	Echo
}

func NewEchoRequest(version uint8, xid uint32, data []byte) *EchoRequest {
	var length uint16 = 8
	if data != nil {
		length += uint16(len(data))
	}

	return &EchoRequest{
		Echo{
			header: Header{
				Version: version,
				Type:    0x02, // OFPT_ECHO_REQUEST
				Length:  length,
				XID:     xid,
			},
			Data: data,
		},
	}
}

type EchoReply struct {
	Echo
}

func NewEchoReply(version uint8, xid uint32, data []byte) *EchoReply {
	var length uint16 = 8
	if data != nil {
		length += uint16(len(data))
	}

	return &EchoReply{
		Echo{
			header: Header{
				Version: version,
				Type:    0x03, // OFPT_ECHO_REPLY
				Length:  length,
				XID:     xid,
			},
			Data: data,
		},
	}
}
