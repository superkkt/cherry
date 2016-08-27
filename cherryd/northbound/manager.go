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

package northbound

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/superkkt/cherry/cherryd/database"
	"github.com/superkkt/cherry/cherryd/log"
	"github.com/superkkt/cherry/cherryd/network"
	"github.com/superkkt/cherry/cherryd/northbound/app"
	"github.com/superkkt/cherry/cherryd/northbound/app/l2switch"
	"github.com/superkkt/cherry/cherryd/northbound/app/monitor"
	"github.com/superkkt/cherry/cherryd/northbound/app/proxyarp"

	"github.com/dlintw/goconf"
	"github.com/pkg/errors"
)

type EventSender interface {
	SetEventListener(network.EventListener)
}

type application struct {
	instance app.Processor
	enabled  bool
}

type Manager struct {
	mutex      sync.Mutex
	log        log.Logger
	conf       *goconf.ConfigFile
	apps       map[string]*application // Registered applications
	head, tail app.Processor
	db         *database.MySQL
}

func NewManager(conf *goconf.ConfigFile, log log.Logger, db *database.MySQL) (*Manager, error) {
	if conf == nil {
		panic("nil config")
	}
	if log == nil {
		panic("nil logger")
	}

	v := &Manager{
		log:  log,
		conf: conf,
		apps: make(map[string]*application),
		db:   db,
	}
	// Registering north-bound applications
	v.register(l2switch.New(conf, log))
	v.register(proxyarp.New(conf, log, db))
	v.register(monitor.New(conf, log))

	return v, nil
}

func (r *Manager) register(app app.Processor) {
	r.apps[strings.ToUpper(app.Name())] = &application{
		instance: app,
		enabled:  false,
	}
}

// XXX: Caller should lock the mutex before they call this function
func (r *Manager) checkDependencies(appNames []string) error {
	if appNames == nil || len(appNames) == 0 {
		// No dependency
		return nil
	}

	for _, name := range appNames {
		app, ok := r.apps[strings.ToUpper(name)]
		r.log.Debug(fmt.Sprintf("app: %+v, ok: %v", app, ok))
		if !ok || !app.enabled {
			return fmt.Errorf("%v application is not loaded", name)
		}
	}

	return nil
}

func (r *Manager) Enable(appName string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.log.Debug(fmt.Sprintf("Enabling %v application..", appName))
	v, ok := r.apps[strings.ToUpper(appName)]
	if !ok {
		return fmt.Errorf("unknown application: %v", appName)
	}
	app := v.instance

	if err := app.Init(); err != nil {
		return errors.Wrap(err, "initializing application")
	}
	if err := r.checkDependencies(app.Dependencies()); err != nil {
		return errors.Wrap(err, "checking dependencies")
	}
	v.enabled = true
	r.log.Debug(fmt.Sprintf("Enabled %v application..", appName))

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
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.head == nil {
		return
	}
	sender.SetEventListener(r.head)
}

func (r *Manager) String() string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var buf bytes.Buffer
	app := r.head
	for app != nil {
		buf.WriteString(fmt.Sprintf("%v\n", app))
		next, ok := app.Next()
		if !ok {
			break
		}
		app = next
	}

	return buf.String()
}
