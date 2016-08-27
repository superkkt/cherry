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

package log

import (
	"log/syslog"
)

type Level uint8

const (
	Debug Level = iota
	Info
	Notice
	Warning
	Error
)

type Logger interface {
	Debug(m string) (err error)
	Err(m string) (err error)
	Info(m string) (err error)
	Notice(m string) (err error)
	Warning(m string) (err error)
}

type Syslog struct {
	writer *syslog.Writer
	level  Level
}

func NewSyslog(l Level) (*Syslog, error) {
	log, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "cherry")
	if err != nil {
		return nil, err
	}

	return &Syslog{
		writer: log,
		level:  l,
	}, nil
}

func (r *Syslog) Debug(m string) (err error) {
	if r.level > Debug {
		return nil
	}

	return r.writer.Debug(m)
}

func (r *Syslog) Err(m string) (err error) {
	if r.level > Error {
		return nil
	}

	return r.writer.Err(m)
}

func (r *Syslog) Info(m string) (err error) {
	if r.level > Info {
		return nil
	}

	return r.writer.Info(m)
}

func (r *Syslog) Notice(m string) (err error) {
	if r.level > Notice {
		return nil
	}

	return r.writer.Notice(m)
}

func (r *Syslog) Warning(m string) (err error) {
	if r.level > Warning {
		return nil
	}

	return r.writer.Warning(m)
}
