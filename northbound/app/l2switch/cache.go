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
	"time"

	"github.com/hashicorp/golang-lru"
)

type flowCache struct {
	cache *lru.Cache
}

func newFlowCache() *flowCache {
	c, err := lru.New(8192)
	if err != nil {
		panic(fmt.Sprintf("LRU flow cache: %v", err))
	}

	return &flowCache{
		cache: c,
	}
}

func (r *flowCache) getKeyString(flow flowParam) string {
	return fmt.Sprintf("%v/%v/%v", flow.device.ID(), flow.dstMAC, flow.outPort)
}

func (r *flowCache) exist(flow flowParam) bool {
	v, ok := r.cache.Get(r.getKeyString(flow))
	if !ok {
		return false
	}
	// Timeout?
	if time.Since(v.(time.Time)) > 5*time.Second {
		return false
	}

	return true
}

func (r *flowCache) add(flow flowParam) {
	// Update if the key already exists
	r.cache.Add(r.getKeyString(flow), time.Now())
}
