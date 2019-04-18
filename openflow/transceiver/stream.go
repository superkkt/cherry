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

package transceiver

import (
	"bufio"
	"io"
	"net"
	"sync"
	"time"
)

// Stream is a buffered I/O channel.
type Stream struct {
	// Underlying socket.
	channel io.ReadWriteCloser

	reader struct {
		mutex sync.Mutex
		// Buffered reader on the underlying socket.
		//
		// NOTE:
		// rd needs locking, otherwise Peek()'s result slice can be
		// corrupted by subsequent Read() or ReadN() calls because
		// the result slice is just a pointer to the reader's internal
		// buffer, which will be overwritten by Read() and ReadN().
		rd        *bufio.Reader
		timeout   time.Duration
		timestamp time.Time
	}

	writer struct {
		mutex     sync.Mutex
		wr        io.Writer
		timeout   time.Duration
		timestamp time.Time
	}

}

type deadline interface {
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

// NewStream returns a new buffered I/O channel. channel is an underlying I/O channel that implements io.ReadWriteCloser.
func NewStream(channel io.ReadWriteCloser, bufSize int) *Stream {
	c := new(Stream)
	c.channel = channel
	c.reader.rd = bufio.NewReaderSize(channel, bufSize)
	c.writer.wr = channel

	return c
}

type dummyAddr struct{}

func (r dummyAddr) Network() string {
	return "DummyAddress"
}

func (r dummyAddr) String() string {
	return ""
}

func (r *Stream) RemoteAddr() net.Addr {
	type addr interface {
		RemoteAddr() net.Addr
	}

	v, ok := r.channel.(addr)
	if !ok {
		return dummyAddr{}
	}

	return v.RemoteAddr()
}

// SetReadTimeout sets read timeout of the underlying socket if it implements deadline interface.
func (r *Stream) SetReadTimeout(t time.Duration) {
	r.reader.mutex.Lock()
	defer r.reader.mutex.Unlock()

	r.reader.timeout = t
	logger.Debugf("set read timeout to %v", t)
}

func (r *Stream) GetReadTimeout() time.Duration {
	r.reader.mutex.Lock()
	defer r.reader.mutex.Unlock()

	return r.reader.timeout
}

// SetWriteTimeout sets write timeout of the underlying socket if it implements deadline interface.
func (r *Stream) SetWriteTimeout(t time.Duration) {
	r.writer.mutex.Lock()
	defer r.writer.mutex.Unlock()

	r.writer.timeout = t
	logger.Debugf("set write timeout to %v", t)
}

func (r *Stream) GetWriteTimeout() time.Duration {
	r.writer.mutex.Lock()
	defer r.writer.mutex.Unlock()

	return r.writer.timeout
}

// Read is a wrapper function of bufio.Reader.Read().
func (r *Stream) Read(p []byte) (n int, err error) {
	r.reader.mutex.Lock()
	defer r.reader.mutex.Unlock()

	r.setReadDeadline()
	n, err = r.reader.rd.Read(p)
	if err != nil {
		return n, err
	}
	r.reader.timestamp = time.Now()

	return n, nil
}

// NOTE: The caller should lock the reader mutex before calling this function.
func (r *Stream) setReadDeadline() {
	// Directly use the underlying socket, instead of the reader, to set I/O timeout.
	d, ok := r.channel.(deadline)
	if !ok {
		logger.Debug("socket does not support the read deadline interface!")
		return
	}

	if r.reader.timeout > 0 {
		d.SetReadDeadline(time.Now().Add(r.reader.timeout))
	} else {
		d.SetReadDeadline(time.Time{})
	}
}

// Peek is a wrapper function of bufio.Reader.Peek().
func (r *Stream) Peek(n int) ([]byte, error) {
	r.reader.mutex.Lock()
	defer r.reader.mutex.Unlock()

	if n <= 0 {
		return []byte{}, nil
	}

	r.setReadDeadline()
	v, err := r.reader.rd.Peek(n)
	if err != nil {
		return nil, err
	}

	// Deep copy of the peek result because v is a pointer to reader's internal
	// buffer that may be corrupted by subsequent other read calls.
	p := make([]byte, len(v))
	copy(p, v)

	return p, nil
}

// ReadN reads exactly n bytes from the underlying socket. It returns non-nil error if len(p) < n, and the data, whose
// length is len(p) bytes long, still remains in the socket buffer.
func (r *Stream) ReadN(n int) (p []byte, err error) {
	r.reader.mutex.Lock()
	defer r.reader.mutex.Unlock()

	r.setReadDeadline()

	// Wait until we have n-bytes data in the reader or timeout.
	if _, err = r.reader.rd.Peek(n); err != nil {
		return nil, err
	}

	p = make([]byte, n)
	c, err := r.reader.rd.Read(p)
	if err != nil {
		return nil, err
	}
	if c != n {
		panic("insufficient read")
	}
	r.reader.timestamp = time.Now()

	return p, nil
}

// LastRead returns the timestamp of the last successful read operation except Peek().
func (r *Stream) LastRead() time.Time {
	r.reader.mutex.Lock()
	defer r.reader.mutex.Unlock()

	return r.reader.timestamp
}

// Write is a wrapper function of net.Conn.Write().
func (r *Stream) Write(p []byte) (n int, err error) {
	r.writer.mutex.Lock()
	defer r.writer.mutex.Unlock()

	r.setWriteDeadline()
	n, err = r.writer.wr.Write(p)
	if err != nil {
		return n, err
	}
	r.writer.timestamp = time.Now()

	return n, nil
}

// NOTE: The caller should lock the writer mutex before calling this function.
func (r *Stream) setWriteDeadline() {
	// Directly use the underlying socket, instead of the writer, to set I/O timeout.
	d, ok := r.channel.(deadline)
	if !ok {
		logger.Debug("socket does not support the write deadline interface!")
		return
	}

	if r.writer.timeout > 0 {
		d.SetWriteDeadline(time.Now().Add(r.writer.timeout))
	} else {
		d.SetWriteDeadline(time.Time{})
	}
}

// LastWrite returns the timestamp of the last successful write operation.
func (r *Stream) LastWrite() time.Time {
	r.writer.mutex.Lock()
	defer r.writer.mutex.Unlock()

	return r.writer.timestamp
}

// Close is a wrapper function of net.Conn.Close().
func (r *Stream) Close() error {
	return r.channel.Close()
}