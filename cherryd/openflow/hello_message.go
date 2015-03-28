/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

type HelloMessage struct {
	Header
	// This message does not contain a body beyond the header
}

func (r *HelloMessage) MarshalBinary() ([]byte, error) {
	r.Header.Length = 8
	return r.Header.MarshalBinary()
}

func (r *HelloMessage) UnmarshalBinary(data []byte) error {
	return r.Header.UnmarshalBinary(data)
}
