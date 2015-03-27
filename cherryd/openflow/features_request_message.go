/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

type FeaturesRequestMessage struct {
	Header
	// This message does not contain a body beyond the header
}

func (r *FeaturesRequestMessage) MarshalBinary() ([]byte, error) {
	return r.Header.MarshalBinary()
}

func (r *FeaturesRequestMessage) UnmarshalBinary(data []byte) error {
	return r.Header.UnmarshalBinary(data)
}
