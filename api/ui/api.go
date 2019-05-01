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
	"errors"
	"net"
	"time"

	"github.com/superkkt/cherry/api"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/superkkt/go-logging"
)

var (
	logger = logging.MustGetLogger("ui")
)

type API struct {
	api.Server
	DB Database

	session *session
}

type Database interface {
	// Exec executes all queries of f in a single transaction. f should return the error raised from the Transaction
	// without any change or wrapping it for deadlock protection.
	Exec(f func(Transaction) error) error
}

type Transaction interface {
	UserTransaction
	GroupTransaction
	SwitchTransaction
	NetworkTransaction
	IPTransaction
	HostTransaction
	VIPTransaction
}

func (r *API) Serve() error {
	if r.DB == nil {
		return errors.New("nil DB")
	}
	r.session = newSession(256, 2*time.Hour)

	return r.Server.Serve(
		rest.Post("/api/v1/user/login", api.ResponseHandler(r.login)),
		rest.Post("/api/v1/user/logout", api.ResponseHandler(r.logout)),
		rest.Post("/api/v1/user/list", api.ResponseHandler(r.listUser)),
		rest.Post("/api/v1/user/add", api.ResponseHandler(r.addUser)),
		rest.Post("/api/v1/user/update", api.ResponseHandler(r.updateUser)),
		rest.Post("/api/v1/user/activate", api.ResponseHandler(r.activateUser)),
		rest.Post("/api/v1/user/deactivate", api.ResponseHandler(r.deactivateUser)),
		rest.Post("/api/v1/group/list", r.listGroup),
		rest.Post("/api/v1/group/add", r.addGroup),
		rest.Post("/api/v1/group/update", r.updateGroup),
		rest.Post("/api/v1/group/remove", r.removeGroup),
		rest.Post("/api/v1/switch/list", r.listSwitch),
		rest.Post("/api/v1/switch/add", r.addSwitch),
		rest.Post("/api/v1/switch/remove", r.removeSwitch),
		rest.Post("/api/v1/network/list", r.listNetwork),
		rest.Post("/api/v1/network/add", r.addNetwork),
		rest.Post("/api/v1/network/remove", r.removeNetwork),
		rest.Post("/api/v1/network/ip", r.listIP),
		rest.Post("/api/v1/host/add", r.addHost),
		rest.Post("/api/v1/host/update", r.updateHost),
		rest.Post("/api/v1/host/activate", r.activateHost),
		rest.Post("/api/v1/host/deactivate", r.deactivateHost),
		rest.Post("/api/v1/host/remove", r.removeHost),
		rest.Post("/api/v1/vip/list", r.listVIP),
		rest.Post("/api/v1/vip/add", r.addVIP),
		rest.Post("/api/v1/vip/remove", r.removeVIP),
		rest.Post("/api/v1/vip/toggle", r.toggleVIP),
	)
}

func (r *API) validateAdminSession(sessionID string) bool {
	session, ok := r.session.Get(sessionID)
	if ok == false {
		return false
	}

	return session.(*User).Admin
}

func (r *API) announce(cidr, mac string) error {
	i, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	m, err := net.ParseMAC(mac)
	if err != nil {
		return err
	}

	logger.Debugf("sending ARP announcement to all hosts to update their ARP caches: ip=%v, mac=%v", i, m)
	if err := r.Controller.Announce(i, m); err != nil {
		// Ignore this error.
		logger.Errorf("failed to send ARP announcement: %v", err)
	} else {
		logger.Debugf("updated all hosts ARP caches: ip=%v, mac=%v", i, m)
	}

	return nil
}
