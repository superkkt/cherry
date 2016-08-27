/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service, Inc. All rights reserved.
 * Kitae Kim <superkkt@sds.co.kr>
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along
 * with this program; if not, write to the Free Software Foundation, Inc.,
 * 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
 */

package trans

import (
	"bufio"
	"io"
	"time"
)

// Stream is a buffered I/O channel.
type Stream struct {
	channel      io.ReadWriteCloser
	reader       *bufio.Reader
	readTimeout  time.Duration
	writeTimeout time.Duration
}

type Deadline interface {
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

// NewStream returns a new buffered I/O channel. channel is an underlying I/O channel that implements io.ReadWriteCloser.
func NewStream(channel io.ReadWriteCloser) *Stream {
	r := bufio.NewReaderSize(channel, 0xFFFF)
	return &Stream{
		channel: channel,
		reader:  r,
	}
}

// SetReadTimeout sets read timeout of the underlying I/O channel if the channel implements Deadline interface.
func (r *Stream) SetReadTimeout(t time.Duration) {
	r.readTimeout = t
}

// SetWriteTimeout sets write timeout of the underlying I/O channel if the channel implements Deadline interface.
func (r *Stream) SetWriteTimeout(t time.Duration) {
	r.writeTimeout = t
}

// Read is a wrapper function of bufio.Reader.Read().
func (r *Stream) Read(p []byte) (n int, err error) {
	if r.readTimeout > 0 {
		d, ok := r.channel.(Deadline)
		if ok {
			d.SetReadDeadline(time.Now().Add(r.readTimeout))
			defer d.SetReadDeadline(time.Time{})
		}
	}

	return r.reader.Read(p)
}

// Peek is a wrapper function of bufio.Reader.Peek().
func (r *Stream) Peek(n int) (p []byte, err error) {
	if r.readTimeout > 0 {
		d, ok := r.channel.(Deadline)
		if ok {
			d.SetReadDeadline(time.Now().Add(r.readTimeout))
			defer d.SetReadDeadline(time.Time{})
		}
	}

	return r.reader.Peek(n)
}

// ReadN reads exactly n bytes from this socket. It returns non-nil error if len(p) < n,
// and the data, whose length is len(p) bytes long, still remains in the socket buffer.
func (r *Stream) ReadN(n int) (p []byte, err error) {
	if r.readTimeout > 0 {
		d, ok := r.channel.(Deadline)
		if ok {
			d.SetReadDeadline(time.Now().Add(r.readTimeout))
			defer d.SetReadDeadline(time.Time{})
		}
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
		d, ok := r.channel.(Deadline)
		if ok {
			d.SetWriteDeadline(time.Now().Add(r.writeTimeout))
			defer d.SetWriteDeadline(time.Time{})
		}
	}

	return r.channel.Write(p)
}

// Close is a wrapper function of net.Conn.Close().
func (r *Stream) Close() error {
	return r.channel.Close()
}
