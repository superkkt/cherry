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
	"sort"
	"strings"
	"sync"
)

var Pool *Application

func init() {
	Pool = &Application{
		p: make([]processor, 0),
	}
}

type processor interface {
	name() string
	priority() uint
	setPriority(uint)
	enable()
	disable()
	enabled() bool
	run(eth *protocol.Ethernet, ingress controller.Point) (drop bool, err error)
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
	mutex sync.Mutex
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

func (r *Application) Run(eth *protocol.Ethernet, ingress controller.Point) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if eth == nil {
		panic("nil Ethernet parameter")
	}

	for _, v := range r.p {
		if !v.enabled() {
			continue
		}
		// XXX: debugging
		fmt.Printf("Running %v (priority=%v)..\n", v.name(), v.priority())
		drop, err := v.run(eth, ingress)
		if err != nil {
			return err
		}
		if drop {
			break
		}
	}

	return nil
}
