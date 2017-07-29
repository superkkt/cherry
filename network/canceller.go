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
	"context"
	"sync"
)

var sessionCanceller *canceller = &canceller{elems: make(map[string]context.CancelFunc)}

type canceller struct {
	mu    sync.Mutex
	elems map[string]context.CancelFunc
}

func pushCanceller(dpid string, canceller context.CancelFunc) {
	sessionCanceller.mu.Lock()
	defer sessionCanceller.mu.Unlock()

	sessionCanceller.elems[dpid] = canceller
}

func popCanceller(dpid string) (cancel context.CancelFunc, ok bool) {
	sessionCanceller.mu.Lock()
	defer sessionCanceller.mu.Unlock()

	cancel, ok = sessionCanceller.elems[dpid]
	if !ok {
		return nil, false
	}
	delete(sessionCanceller.elems, dpid)

	return cancel, true
}
