/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package app

import (
	"git.sds.co.kr/cherry.git/cherryd/internal/network"
	"git.sds.co.kr/cherry.git/cherryd/protocol"
)

// Processor should prepare to be executed by multiple goroutines simultaneously.
type Processor interface {
	network.EventListener
	Init() error
	// Name returns the application name that is globally unique
	Name() string
	Next() (next Processor, ok bool)
	SetNext(Processor)
}

type BaseProcessor struct {
	next Processor
}

func (r *BaseProcessor) Name() string {
	return "BaseProcessor"
}

func (r *BaseProcessor) OnPacketIn(finder network.Finder, ingress *network.Port, eth *protocol.Ethernet) error {
	// Do nothging and execute the next processor if it exists
	next, ok := r.Next()
	if !ok {
		return nil
	}
	return next.OnPacketIn(finder, ingress, eth)
}

func (r *BaseProcessor) OnDeviceUp(finder network.Finder, device *network.Device) error {
	// Do nothging and execute the next processor if it exists
	next, ok := r.Next()
	if !ok {
		return nil
	}
	return next.OnDeviceUp(finder, device)
}

func (r *BaseProcessor) OnDeviceDown(finder network.Finder, device *network.Device) error {
	// Do nothging and execute the next processor if it exists
	next, ok := r.Next()
	if !ok {
		return nil
	}
	return next.OnDeviceDown(finder, device)
}

func (r *BaseProcessor) OnPortUp(finder network.Finder, port *network.Port) error {
	// Do nothging and execute the next processor if it exists
	next, ok := r.Next()
	if !ok {
		return nil
	}
	return next.OnPortUp(finder, port)
}

func (r *BaseProcessor) OnPortDown(finder network.Finder, port *network.Port) error {
	// Do nothging and execute the next processor if it exists
	next, ok := r.Next()
	if !ok {
		return nil
	}
	return next.OnPortDown(finder, port)
}

func (r *BaseProcessor) OnTopologyChange(finder network.Finder) error {
	// Do nothging and execute the next processor if it exists
	next, ok := r.Next()
	if !ok {
		return nil
	}
	return next.OnTopologyChange(finder)
}

func (r *BaseProcessor) Next() (next Processor, ok bool) {
	if r.next != nil {
		return r.next, true
	}

	return nil, false
}

func (r *BaseProcessor) SetNext(next Processor) {
	r.next = next
}
