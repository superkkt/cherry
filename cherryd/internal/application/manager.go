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
	"git.sds.co.kr/cherry.git/cherryd/protocol"
	"strings"
)

type PacketHandler interface {
	Name() string
	// processPacket should prepare to be executed simultaneously by multiple goroutines.
	Process(*protocol.Ethernet, *network.Port, log.Logger) (drop bool, err error)
}

type Manager struct {
	registered map[string]PacketHandler // Registered applications
	enabled    []PacketHandler          // Enabled applications
	log        log.Logger
}

func NewManager(log log.Logger) *Manager {
	v := &Manager{
		registered: make(map[string]PacketHandler),
		enabled:    make([]PacketHandler, 0),
		log:        log,
	}
	// Registering north-bound applications
	v.register(l2switch.New())

	return v
}

func (r *Manager) register(app PacketHandler) {
	r.registered[strings.ToUpper(app.Name())] = app
}

func (r *Manager) Enable(appName string) {
	v, ok := r.registered[strings.ToUpper(appName)]
	if !ok {
		return
	}
	r.enabled = append(r.enabled, v)
}

func (r *Manager) Process(eth *protocol.Ethernet, ingress *network.Port) error {
	for _, v := range r.enabled {
		drop, err := v.Process(eth, ingress, r.log)
		if drop || err != nil {
			return err
		}
	}

	return nil
}
