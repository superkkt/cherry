/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package openflow

import (
	"bufio"
	"net"
	"time"
)

// Stream is a buffered socket connection.
type Stream struct {
	conn   net.Conn
	reader *bufio.Reader
}

// NewStream returns a new buffered connection. conn is an already connected connection.
func NewStream(conn net.Conn) *Stream {
	r := bufio.NewReaderSize(conn, 0xFFFF)
	return &Stream{
		conn:   conn,
		reader: r,
	}
}

// setDeadline is a wrapper function of net.Conn.SetDeadline().
func (r *Stream) setDeadline(deadline time.Time) {
	r.conn.SetDeadline(deadline)
}

// read is a wrapper function of bufio.Reader.Read().
func (r *Stream) read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

// peek is a wrapper function of bufio.Reader.Peek().
func (r *Stream) peek(n int) (p []byte, err error) {
	return r.reader.Peek(n)
}

// readN reads exactly n bytes from this socket. It returns non-nil error
// if len(p) < n, and the data, whose length is len(p) bytes long, still remains in the
// socket buffer.
func (r *Stream) readN(n int) (p []byte, err error) {
	p = make([]byte, n)
	if _, err = r.reader.Peek(n); err != nil {
		return nil, err
	}
	c, err := r.reader.Read(p)
	if c != n {
		panic("insufficient read")
	}
	if err != nil {
		return nil, err
	}

	return p, nil
}

// write is a wrapper function of net.Conn.Write().
func (r *Stream) write(p []byte) (n int, err error) {
	return r.conn.Write(p)
}

// shutdown is a wrapper function of net.Conn.Close().
func (r *Stream) shutdown() error {
	return r.conn.Close()
}
