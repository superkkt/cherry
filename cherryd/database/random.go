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

package database

import (
	"math/rand"
	"sync"
)

// randomSourece is safe for concurrent use by multiple goroutines.
type randomSource struct {
	sync.Mutex
	src rand.Source
}

func (r *randomSource) Int63() (n int64) {
	r.Lock()
	defer r.Unlock()

	return r.src.Int63()
}

func (r *randomSource) Seed(seed int64) {
	r.Lock()
	defer r.Unlock()

	r.src.Seed(seed)
}
