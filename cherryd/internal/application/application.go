/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package application

import (
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/internal/controller"
	"git.sds.co.kr/cherry.git/cherryd/net/protocol"
	"git.sds.co.kr/cherry.git/cherryd/openflow"
	"net"
	"sort"
	"strings"
	"sync"
)

var Pool *Application
var virtualMAC net.HardwareAddr

func init() {
	Pool = &Application{
		p: make([]processor, 0),
	}

	// Locally Administered Address
	mac, err := net.ParseMAC("02:DB:CA:FE:00:01")
	if err != nil {
		panic("invalid MAC address")
	}
	virtualMAC = mac
}

type processor interface {
	name() string
	priority() uint
	setPriority(uint)
	enable()
	disable()
	enabled() bool
}

type packetHandler interface {
	// processPacket should prepare to be executed simultaneously by multiple goroutines.
	processPacket(eth *protocol.Ethernet, ingress controller.Point) (drop bool, err error)
}

type eventHandler interface {
	// processEvent should prepare to be executed simultaneously by multiple goroutines.
	processEvent(device *controller.Device, port openflow.Port, mac []net.HardwareAddr) error
}

type baseProcessor struct {
	prior uint
	state bool
}

func (r baseProcessor) priority() uint {
	return r.prior
}

func (r *baseProcessor) setPriority(p uint) {
	r.prior = p
}

func (r *baseProcessor) enable() {
	r.state = true
}

func (r *baseProcessor) disable() {
	r.state = false
}

func (r baseProcessor) enabled() bool {
	return r.state
}

type sortByPriority []processor

func (r sortByPriority) Len() int {
	return len(r)
}

func (r sortByPriority) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r sortByPriority) Less(i, j int) bool {
	return r[i].priority() < r[j].priority()
}

type Application struct {
	mutex sync.RWMutex
	p     []processor
}

func (r *Application) add(p processor) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	p.disable()
	r.p = append(r.p, p)
	sort.Sort(sortByPriority(r.p))
}

func (r *Application) Enable(name string, priority uint) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	enabled := false
	for _, v := range r.p {
		if strings.ToUpper(v.name()) != strings.ToUpper(name) {
			continue
		}
		v.setPriority(priority)
		v.enable()
		enabled = true
		break
	}
	if !enabled {
		return fmt.Errorf("not found application %v", name)
	}
	sort.Sort(sortByPriority(r.p))

	return nil
}

func (r *Application) ProcessPacket(eth *protocol.Ethernet, ingress controller.Point) error {
	// Lock for reading
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if eth == nil {
		panic("nil Ethernet parameter")
	}

	for _, v := range r.p {
		handler, ok := v.(packetHandler)
		if !v.enabled() || !ok {
			continue
		}
		// XXX: debugging
		fmt.Printf("Running %v (priority=%v)..\n", v.name(), v.priority())
		drop, err := handler.processPacket(eth, ingress)
		if err != nil {
			return err
		}
		if drop {
			break
		}
	}

	return nil
}

func (r *Application) ProcessEvent(device *controller.Device, port openflow.Port, mac []net.HardwareAddr) error {
	// Lock for reading
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, v := range r.p {
		handler, ok := v.(eventHandler)
		if !v.enabled() || !ok {
			continue
		}
		if err := handler.processEvent(device, port, mac); err != nil {
			return err
		}
	}

	return nil
}
