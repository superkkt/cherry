/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
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

package northbound

import (
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"git.sds.co.kr/cherry.git/cherryd/internal/network"
	"git.sds.co.kr/cherry.git/cherryd/internal/northbound/app"
	"git.sds.co.kr/cherry.git/cherryd/internal/northbound/app/firewall"
	"git.sds.co.kr/cherry.git/cherryd/internal/northbound/app/l2switch"
	"git.sds.co.kr/cherry.git/cherryd/internal/northbound/app/lb"
	"git.sds.co.kr/cherry.git/cherryd/internal/northbound/app/proxyarp"
	"git.sds.co.kr/cherry.git/cherryd/internal/northbound/app/router"
	"github.com/dlintw/goconf"
	"strings"
)

type EventSender interface {
	SetEventListener(network.EventListener)
}

type Manager struct {
	log        log.Logger
	conf       *goconf.ConfigFile
	apps       map[string]app.Processor // Registered applications
	head, tail app.Processor
}

func NewManager(conf *goconf.ConfigFile, log log.Logger) *Manager {
	if conf == nil {
		panic("nil config")
	}
	if log == nil {
		panic("nil logger")
	}

	v := &Manager{
		log:  log,
		conf: conf,
		apps: make(map[string]app.Processor),
	}
	// Registering north-bound applications
	v.register(l2switch.New(conf, log))
	v.register(proxyarp.New(conf, log))
	v.register(lb.New(conf, log))
	v.register(router.New(conf, log))
	v.register(firewall.New(conf, log))

	return v
}

func (r *Manager) register(app app.Processor) {
	r.apps[strings.ToUpper(app.Name())] = app
}

func (r *Manager) Enable(appName string) error {
	r.log.Debug(fmt.Sprintf("Enabling %v application..", appName))

	app, ok := r.apps[strings.ToUpper(appName)]
	if !ok {
		return fmt.Errorf("unknown application: %v", appName)
	}
	if err := app.Init(); err != nil {
		return err
	}

	if r.head == nil {
		r.head = app
		r.tail = app
		return nil
	}
	r.tail.SetNext(app)
	r.tail = app

	return nil
}

func (r *Manager) AddEventSender(sender EventSender) {
	if r.head == nil {
		return
	}
	sender.SetEventListener(r.head)
}
