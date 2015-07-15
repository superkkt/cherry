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

package lb

import (
	"fmt"
	"github.com/dlintw/goconf"
	"github.com/superkkt/cherry/cherryd/log"
	"github.com/superkkt/cherry/cherryd/northbound/app"
)

type LoadBalancer struct {
	app.BaseProcessor
	conf *goconf.ConfigFile
	log  log.Logger
	db   database
}

type database interface{}

func New(conf *goconf.ConfigFile, log log.Logger, db database) *LoadBalancer {
	return &LoadBalancer{
		conf: conf,
		log:  log,
		db:   db,
	}
}

func (r *LoadBalancer) Name() string {
	return "LoadBalancer"
}

func (r *LoadBalancer) String() string {
	return fmt.Sprintf("%v", r.Name())
}
