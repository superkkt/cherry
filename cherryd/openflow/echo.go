/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

type Echo struct {
	Message
	Data []byte
}

func (r *Echo) MarshalBinary() ([]byte, error) {
	r.SetPayload(r.Data)
	return r.Message.MarshalBinary()
}

func (r *Echo) UnmarshalBinary(data []byte) error {
	if err := r.Message.UnmarshalBinary(data); err != nil {
		return err
	}
	r.Data = r.Payload()

	return nil
}

type EchoRequest struct {
	Echo
}

func NewEchoRequest(version uint8, xid uint32, data []byte) *EchoRequest {
	return &EchoRequest{
		Echo{
			// OFPT_ECHO_REQUEST
			Message: NewMessage(version, 0x02, xid),
			Data:    data,
		},
	}
}

type EchoReply struct {
	Echo
}

func NewEchoReply(version uint8, xid uint32, data []byte) *EchoReply {
	return &EchoReply{
		Echo{
			// OFPT_ECHO_REPLY
			Message: NewMessage(version, 0x03, xid),
			Data:    data,
		},
	}
}
