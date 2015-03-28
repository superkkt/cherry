/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

type PortStatusMessage struct {
	Header
	Reason uint8
	Target Port
}

func (r *PortStatusMessage) UnmarshalBinary(data []byte) error {
	header := &Header{}
	if err := header.UnmarshalBinary(data); err != nil {
		return err
	}
	if len(data) != int(header.Length) {
		return ErrInvalidPacketLength
	}

	r.Header = *header
	r.Reason = data[8]
	if err := r.Target.UnmarshalBinary(data[16:64]); err != nil {
		return err
	}

	return nil
}
