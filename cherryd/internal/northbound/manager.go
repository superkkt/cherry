/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package northbound

import (
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"git.sds.co.kr/cherry.git/cherryd/internal/network"
	"git.sds.co.kr/cherry.git/cherryd/internal/northbound/app"
	"git.sds.co.kr/cherry.git/cherryd/internal/northbound/app/l2switch"
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

	return v
}

func (r *Manager) register(app app.Processor) {
	r.apps[strings.ToUpper(app.Name())] = app
}

func (r *Manager) Enable(appName string) {
	app, ok := r.apps[strings.ToUpper(appName)]
	if !ok {
		return
	}

	if r.head == nil {
		r.head = app
		r.tail = app
		return
	}
	r.tail.SetNext(app)
	r.tail = app
}

func (r *Manager) AddEventSender(sender EventSender) {
	if r.head == nil {
		return
	}
	sender.SetEventListener(r.head)
}
