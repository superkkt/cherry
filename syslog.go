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

package main

import (
	"fmt"
	slog "log/syslog"
	"runtime"
	"strings"

	"github.com/op/go-logging"
)

type syslog struct {
	writer *slog.Writer
}

func newSyslog(prefix string) (logging.Backend, error) {
	w, err := slog.New(slog.LOG_CRIT, prefix)
	if err != nil {
		return nil, err
	}

	return &syslog{writer: w}, nil
}

func (r *syslog) Log(level logging.Level, calldepth int, record *logging.Record) error {
	line := fmt.Sprintf("%v (TID=%v)", record.Formatted(calldepth+1), getGoRoutineID())
	switch level {
	case logging.CRITICAL:
		return r.writer.Crit(line)
	case logging.ERROR:
		return r.writer.Err(line)
	case logging.WARNING:
		return r.writer.Warning(line)
	case logging.NOTICE:
		return r.writer.Notice(line)
	case logging.INFO:
		return r.writer.Info(line)
	case logging.DEBUG:
		return r.writer.Debug(line)
	default:
		panic("unexpected log level")
	}
}

func getGoRoutineID() string {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	return strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
}
