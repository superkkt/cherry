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

package network

import (
	"fmt"
	"time"

	"github.com/superkkt/cherry/openflow"

	lru "github.com/hashicorp/golang-lru"
)

type flowCache struct {
	cache      *lru.Cache
	expiration time.Duration
}

func newFlowCache(expiration time.Duration) *flowCache {
	c, err := lru.New(8192)
	if err != nil {
		panic(fmt.Sprintf("failed to init a LRU flow cache: %v", err))
	}

	return &flowCache{
		cache:      c,
		expiration: expiration,
	}
}

func (r *flowCache) Add(match openflow.Match, port openflow.OutPort) error {
	key, err := r.key(match, port)
	if err != nil {
		return err
	}

	t := time.Now()
	// Update if the key already exists.
	r.cache.Add(key, t)
	logger.Debugf("added a new flow cache: key=%v, timestamp=%v", key, t)

	return nil
}

func (r *flowCache) key(match openflow.Match, port openflow.OutPort) (string, error) {
	m, err := match.MarshalBinary()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v/%v", m, port), nil
}

func (r *flowCache) InProgress(match openflow.Match, port openflow.OutPort) (ok bool, err error) {
	key, err := r.key(match, port)
	if err != nil {
		return false, err
	}

	v, ok := r.cache.Get(key)
	if !ok {
		return false, nil
	}
	timestamp := v.(time.Time)

	// Timeout?
	if time.Since(timestamp) > r.expiration {
		r.cache.Remove(key)
		logger.Debugf("removed the timed-out flow cache: key=%v", key)
		return false, nil
	}

	return true, nil
}

func (r *flowCache) RemoveAll() {
	r.cache.Purge()
	logger.Debug("removed all the flow caches")
}
