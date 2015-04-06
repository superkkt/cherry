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
	conn         net.Conn
	reader       *bufio.Reader
	readTimeout  time.Duration
	writeTimeout time.Duration
}

// NewStream returns a new buffered connection. conn is an already connected connection.
func NewStream(conn net.Conn) *Stream {
	r := bufio.NewReaderSize(conn, 0xFFFF)
	return &Stream{
		conn:   conn,
		reader: r,
	}
}

func (r *Stream) SetReadTimeout(t time.Duration) {
	r.readTimeout = t
}

func (r *Stream) SetWriteTimeout(t time.Duration) {
	r.writeTimeout = t
}

// Read is a wrapper function of bufio.Reader.Read().
func (r *Stream) Read(p []byte) (n int, err error) {
	if r.readTimeout > 0 {
		r.conn.SetReadDeadline(time.Now().Add(r.readTimeout))
		defer r.conn.SetReadDeadline(time.Time{})
	}

	return r.reader.Read(p)
}

// Peek is a wrapper function of bufio.Reader.Peek().
func (r *Stream) Peek(n int) (p []byte, err error) {
	if r.readTimeout > 0 {
		r.conn.SetReadDeadline(time.Now().Add(r.readTimeout))
		defer r.conn.SetReadDeadline(time.Time{})
	}

	return r.reader.Peek(n)
}

// ReadN reads exactly n bytes from this socket. It returns non-nil error
// if len(p) < n, and the data, whose length is len(p) bytes long, still remains in the
// socket buffer.
func (r *Stream) ReadN(n int) (p []byte, err error) {
	if r.readTimeout > 0 {
		r.conn.SetReadDeadline(time.Now().Add(r.readTimeout))
		defer r.conn.SetReadDeadline(time.Time{})
	}

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

// Write is a wrapper function of net.Conn.Write().
func (r *Stream) Write(p []byte) (n int, err error) {
	if r.writeTimeout > 0 {
		r.conn.SetWriteDeadline(time.Now().Add(r.writeTimeout))
		defer r.conn.SetWriteDeadline(time.Time{})
	}

	return r.conn.Write(p)
}

// Close is a wrapper function of net.Conn.Close().
func (r *Stream) Close() error {
	return r.conn.Close()
}
