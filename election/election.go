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

package election

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/superkkt/go-logging"
)

var (
	logger = logging.MustGetLogger("election")
)

const (
	interval = 1 * time.Second
)

type Observer struct {
	uid string
	db  Database

	mutex  sync.Mutex
	master bool
}

type Database interface {
	// Elect selects a new master as uid if there is a no existing master that has
	// been updated within expiration. elected will be true if this uid has been
	// elected as the new master or was already elected.
	Elect(uid string, expiration time.Duration) (elected bool, err error)
}

func New(db Database) *Observer {
	return &Observer{
		uid: generateRandomUID(),
		db:  db,
	}
}

func generateRandomUID() string {
	src := fmt.Sprintf("%v.%v.%v", time.Now().UnixNano(), os.Getpid(), rand.Int63())
	sum := sha256.Sum256([]byte(src))
	return fmt.Sprintf("%x", sum)
}

func (r *Observer) Run(ctx context.Context) error {
	logger.Debugf("starting an election observer: uid=%v", r.uid)

	ticker := time.Tick(interval)
	// Infinite loop.
	for {
		prev := r.getMaster()
		elected, err := r.db.Elect(r.uid, interval*15)
		if err != nil {
			return err
		}
		r.setMaster(elected)
		logger.Debugf("master election result: prev=%v, elected=%v", prev, elected)

		if prev != elected {
			if prev == true {
				// Previous master.
				logger.Fatal("master controller has been changed: demoted from the master: self shutting down to avoid split-brain")
			} else {
				// New master.
				logger.Warning("master controller has been changed: elected as a new master")
			}
		}

		// Wait the context cancels or the ticker rasises.
		select {
		case <-ctx.Done():
			logger.Debug("terminating the election observer...")
			return nil
		case <-ticker:
			// Do nothing.
		}
	}
}

func (r *Observer) IsMaster() bool {
	return r.getMaster()
}

func (r *Observer) setMaster(value bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.master = value
}

func (r *Observer) getMaster() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.master
}
