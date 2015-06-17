/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package application

import (
	"git.sds.co.kr/cherry.git/cherryd/internal/application/l2switch"
	"git.sds.co.kr/cherry.git/cherryd/internal/log"
	"git.sds.co.kr/cherry.git/cherryd/internal/network"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"git.sds.co.kr/cherry.git/cherryd/protocol"
	"github.com/dlintw/goconf"
	"strings"
)

// processor should prepare to be executed simultaneously by multiple goroutines.
type processor interface {
	// Name returns the application name that is globally unique
	Name() string
	ProcessPacket(network.Finder, *protocol.Ethernet, *network.Port) (drop bool, err error)
	ProcessPortChange(network.Finder, *network.Device, openflow.PortStatus) error
	ProcessDeviceClose(network.Finder, *network.Device) error
}

type Manager struct {
	registered map[string]processor // Registered applications
	enabled    []processor          // Enabled applications
	log        log.Logger
	conf       *goconf.ConfigFile
}

func NewManager(conf *goconf.ConfigFile, log log.Logger) *Manager {
	if conf == nil {
		panic("nil config")
	}
	if log == nil {
		panic("nil logger")
	}

	v := &Manager{
		registered: make(map[string]processor),
		enabled:    make([]processor, 0),
		log:        log,
		conf:       conf,
	}
	// Registering north-bound applications
	v.register(l2switch.New(conf, log))

	return v
}

func (r *Manager) register(app processor) {
	r.registered[strings.ToUpper(app.Name())] = app
}

func (r *Manager) Enable(appName string) {
	v, ok := r.registered[strings.ToUpper(appName)]
	if !ok {
		return
	}
	r.enabled = append(r.enabled, v)
}

func (r *Manager) ProcessPacket(finder network.Finder, eth *protocol.Ethernet, ingress *network.Port) error {
	for _, v := range r.enabled {
		drop, err := v.ProcessPacket(finder, eth, ingress)
		if drop || err != nil {
			return err
		}
	}

	return nil
}

func (r *Manager) ProcessPortChange(finder network.Finder, device *network.Device, status openflow.PortStatus) error {
	for _, v := range r.enabled {
		err := v.ProcessPortChange(finder, device, status)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Manager) ProcessDeviceClose(finder network.Finder, device *network.Device) error {
	for _, v := range r.enabled {
		err := v.ProcessDeviceClose(finder, device)
		if err != nil {
			return err
		}
	}

	return nil
}
