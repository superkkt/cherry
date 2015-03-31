/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

type BarrierRequestMessage struct {
	Header
	// This message does not contain a body beyond the header
}

func (r *BarrierRequestMessage) MarshalBinary() ([]byte, error) {
	r.Header.Length = 8
	return r.Header.MarshalBinary()
}

func (r *BarrierRequestMessage) UnmarshalBinary(data []byte) error {
	return r.Header.UnmarshalBinary(data)
}

type BarrierReplyMessage struct {
	Header
	// This message does not contain a body beyond the header
}

func (r *BarrierReplyMessage) MarshalBinary() ([]byte, error) {
	r.Header.Length = 8
	return r.Header.MarshalBinary()
}

func (r *BarrierReplyMessage) UnmarshalBinary(data []byte) error {
	return r.Header.UnmarshalBinary(data)
}
