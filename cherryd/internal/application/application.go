/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package application

import (
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/internal/device"
	"git.sds.co.kr/cherry.git/cherryd/net/protocol"
	"sort"
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
	run(eth *protocol.Ethernet, ingress device.Point) (drop bool, err error)
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

	r.p = append(r.p, p)
	sort.Sort(sortByPriority(r.p))
}

func (r *Application) Run(eth *protocol.Ethernet, ingress device.Point) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if eth == nil {
		panic("nil Ethernet parameter")
	}

	for _, v := range r.p {
		// XXX: debugging
		fmt.Printf("Running %v(%v)..\n", v.name(), v.priority())
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
