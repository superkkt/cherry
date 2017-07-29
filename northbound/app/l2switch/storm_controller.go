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
	"sync"
	"time"

	"github.com/superkkt/cherry/network"
	"github.com/superkkt/logger"
)

type stormController struct {
	mutex      sync.Mutex
	max        uint
	broadcasts []time.Time
	bcaster    broadcaster
}

type broadcaster interface {
	flood(ingress *network.Port, packet []byte) error
}

// max is the number of broadcasts that are allowed per second.
func newStormController(max uint, bcaster broadcaster) *stormController {
	if max <= 0 {
		panic("max should be greater than zero")
	}
	if bcaster == nil {
		panic("bcaster is nil")
	}

	return &stormController{
		max:        max,
		broadcasts: make([]time.Time, 0),
		bcaster:    bcaster,
	}
}

func (r *stormController) broadcast(ingress *network.Port, packet []byte) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	t := time.Now()
	bcasts := append(r.broadcasts, t)
	l := uint(len(bcasts))
	if l <= r.max {
		r.broadcasts = bcasts
		return r.bcaster.flood(ingress, packet)
	}
	// Only allows r.max broadcasts per 1 second
	if t.Sub(bcasts[0]) > 1*time.Second {
		// Shrink (l > r.max)
		r.broadcasts = bcasts[l-r.max : l]
		return r.bcaster.flood(ingress, packet)
	}
	// Deny! r.broadcast should not be updated!
	logger.Info("too many broadcasts: broadcast is denied to avoid the broadcast storm!")

	return nil
}
