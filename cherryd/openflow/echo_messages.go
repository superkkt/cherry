/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

type EchoMessage struct {
	Header
	Data []byte
}

func (r *EchoMessage) MarshalBinary() ([]byte, error) {
	var length uint16 = 8 // header length
	if r.Data != nil {
		length += uint16(len(r.Data))
	}

	r.Header.Length = length
	header, err := r.Header.MarshalBinary()
	if err != nil {
		return nil, err
	}

	v := make([]byte, length)
	copy(v[0:8], header)
	if r.Data != nil && len(r.Data) > 0 {
		copy(v[8:], r.Data)
	}

	return v, nil
}

func (r *EchoMessage) UnmarshalBinary(data []byte) error {
	header := &Header{}
	if err := header.UnmarshalBinary(data); err != nil {
		return err
	}
	if len(data) != int(header.Length) {
		return ErrInvalidPacketLength
	}

	r.Header = *header
	if header.Length > 8 {
		r.Data = make([]byte, header.Length-8)
		copy(r.Data, data[8:])
	}

	return nil
}

type EchoRequestMessage struct {
	EchoMessage
}

type EchoReplyMessage struct {
	EchoMessage
}
