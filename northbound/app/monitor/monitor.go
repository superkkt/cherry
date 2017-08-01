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

package monitor

import (
	"errors"
	"fmt"
	"strings"

	"github.com/superkkt/cherry/network"
	"github.com/superkkt/cherry/northbound/app"

	"github.com/superkkt/go-logging"
	"github.com/superkkt/viper"
)

var (
	logger = logging.MustGetLogger("monitor")
)

type Monitor struct {
	app.BaseProcessor
	email string
}

func New() *Monitor {
	return &Monitor{}
}

func (r *Monitor) Init() error {
	email := viper.GetString("default.admin_email")
	if len(email) == 0 || !strings.Contains(email, "@") {
		return errors.New("invalid admin_email in the config file")
	}
	r.email = email

	return nil
}

func (r *Monitor) Name() string {
	return "Monitor"
}

func (r *Monitor) String() string {
	return fmt.Sprintf("%v", r.Name())
}

func (r *Monitor) OnDeviceUp(finder network.Finder, device *network.Device) error {
	go func() {
		subject := "Cherry: device is up!"
		body := fmt.Sprintf("DPID: %v", device.ID())
		if err := r.sendAlarm(subject, body); err != nil {
			logger.Errorf("failed to send an alarm email: %v", err)
		}
	}()
	logger.Warningf("switch device up: DPID=%v", device.ID())

	return r.BaseProcessor.OnDeviceUp(finder, device)
}

func (r *Monitor) OnDeviceDown(finder network.Finder, device *network.Device) error {
	go func() {
		subject := "Cherry: device is down!"
		body := fmt.Sprintf("DPID: %v", device.ID())
		if err := r.sendAlarm(subject, body); err != nil {
			logger.Errorf("failed to send an alarm email: %v", err)
		}
	}()
	logger.Warningf("switch device down: DPID=%v", device.ID())

	return r.BaseProcessor.OnDeviceDown(finder, device)
}

func (r *Monitor) sendAlarm(subject, body string) error {
	from := "noreply@sds.co.kr"
	to := []string{r.email}
	header := fmt.Sprintf("From: %v\r\nTo: %v\r\nSubject: %v", from, r.email, subject)
	msg := []byte(fmt.Sprintf("%v\r\n\r\n%v", header, body))

	if err := sendmail(from, to, msg); err != nil {
		return err
	}
	logger.Debugf("sent an alarm email to %v: subject=%v", r.email, subject)

	return nil

}
