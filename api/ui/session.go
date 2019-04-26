/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015-2019 Samjung Data Service, Inc. All rights reserved.
 *
 *  Kitae Kim <superkkt@sds.co.kr>
 *  Donam Kim <donam.kim@sds.co.kr>
 *  Jooyoung Kang <jooyoung.kang@sds.co.kr>
 *  Changjin Choi <ccj9707@sds.co.kr>
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

package ui

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/golang-lru"
)

type session struct {
	storage *lru.Cache
	timeout time.Duration
}

func newSession(size int, timeout time.Duration) *session {
	c, err := lru.New(size)
	if err != nil {
		panic(err)
	}

	return &session{
		storage: c,
		timeout: timeout,
	}
}

func (r *session) Add(v interface{}) (id string) {
	src := fmt.Sprintf("%v.%v.%v", spew.Sdump(v), time.Now().UnixNano(), rand.Float64())
	id = strings.ToUpper(hash(src))
	r.storage.Add(id, &sessionEntry{Data: v, Timestamp: time.Now()})

	return id
}

type sessionEntry struct {
	Data      interface{}
	Timestamp time.Time
}

func hash(value string) string {
	h := sha256.New()
	h.Write([]byte(value))

	return hex.EncodeToString(h.Sum(nil))
}

func (r *session) Get(id string) (interface{}, bool) {
	id = strings.ToUpper(id)

	v, ok := r.storage.Get(id)
	if ok == false {
		return nil, false
	}

	e := v.(*sessionEntry)
	if time.Since(e.Timestamp) > r.timeout {
		r.storage.Remove(id)
		return nil, false
	}
	e.Timestamp = time.Now()

	return e.Data, true
}

func (r *session) Remove(id string) bool {
	id = strings.ToUpper(id)

	if r.storage.Contains(id) == false {
		return false
	}
	r.storage.Remove(id)

	return true
}
