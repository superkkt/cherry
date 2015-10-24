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

package l2switch

import (
	"fmt"
	"testing"
	"time"

	"github.com/superkkt/cherry/cherryd/network"
)

func TestStorm(t *testing.T) {
	max := uint(100)
	dummy := new(dummyFlooder)
	storm := newStormController(max, new(dummyLogger), dummy)
	fmt.Printf("%v\n", time.Now())
	for i := uint(0); i < max; i++ {
		storm.broadcast(nil, nil)
		if dummy.getCounter() != uint64(i+1) {
			t.Fatalf("Unexpected flood counter: expected=%v, got=%v", i+1, dummy.getCounter())
		}
	}
	for i := 0; i < 10; i++ {
		fmt.Printf("%v\n", time.Now())
		storm.broadcast(nil, nil)
		if dummy.getCounter() != uint64(max) {
			t.Fatalf("Unexpected flood counter: expected=%v, got=%v", max, dummy.getCounter())
		}
	}
	time.Sleep(1 * time.Second)
	fmt.Printf("%v\n", time.Now())
	for i := uint(0); i < max-1; i++ {
		storm.broadcast(nil, nil)
		if dummy.getCounter() != uint64(max+i+1) {
			t.Fatalf("Unexpected flood counter: expected=%v, got=%v", max+1, dummy.getCounter())
		}
	}
}

func TestPeriodicBroadcast(t *testing.T) {
	max := uint(1)
	dummy := new(dummyFlooder)
	storm := newStormController(max, new(dummyLogger), dummy)
	for i := 0; i < 10; i++ {
		fmt.Printf("Count: %v, Timestamp: %v\n", i, time.Now())
		storm.broadcast(nil, nil)
		if dummy.getCounter() != uint64(i+1) {
			t.Fatalf("Unexpected flood counter: expected=%v, got=%v", i+1, dummy.getCounter())
		}
		time.Sleep(1 * time.Second)
	}
}

func TestPeriodicStorm(t *testing.T) {
	max := uint(1)
	dummy := new(dummyFlooder)
	storm := newStormController(max, new(dummyLogger), dummy)
	for i := 0; i < 10; i++ {
		fmt.Printf("Count: %v, Timestamp: %v\n", i, time.Now())
		storm.broadcast(nil, nil)
		if dummy.getCounter() != uint64(i+1) {
			t.Fatalf("Unexpected flood counter: expected=%v, got=%v", i+1, dummy.getCounter())
		}
		storm.broadcast(nil, nil)
		if dummy.getCounter() != uint64(i+1) {
			t.Fatalf("Unexpected flood counter: expected=%v, got=%v", i+1, dummy.getCounter())
		}
		time.Sleep(1 * time.Second)
	}
}

type dummyFlooder struct {
	counter uint64
}

func (r *dummyFlooder) flood(ingress *network.Port, packet []byte) error {
	r.counter++
	return nil
}

func (r *dummyFlooder) getCounter() uint64 {
	return r.counter
}

type dummyLogger struct{}

func (r *dummyLogger) Debug(m string) (err error) {
	fmt.Printf("DEBUG: %v\n", m)
	return nil
}

func (r *dummyLogger) Err(m string) (err error) {
	fmt.Printf("ERR: %v\n", m)
	return nil
}

func (r *dummyLogger) Info(m string) (err error) {
	fmt.Printf("INFO: %v\n", m)
	return nil
}

func (r *dummyLogger) Notice(m string) (err error) {
	fmt.Printf("NOTICE: %v\n", m)
	return nil
}

func (r *dummyLogger) Warning(m string) (err error) {
	fmt.Printf("WARN: %v\n", m)
	return nil
}
