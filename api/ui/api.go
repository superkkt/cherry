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
	// Auth returns information for a user if name and password match. Otherwise, it returns nil.
	Auth(name, password string) (*User, error)
	Users(offset uint32, limit uint8) ([]User, error)
	AddUser(name, password string) (id uint64, duplicated bool, err error)
	UpdateUser(id uint64, password *string, admin *bool) error
	ActivateUser(id uint64) error
	DeactivateUser(id uint64) error

	Groups(offset uint32, limit uint8) ([]Group, error)
	AddGroup(name string) (id uint64, duplicated bool, err error)
	UpdateGroup(id uint64, name string) (duplicated bool, err error)
	RemoveGroup(id uint64) error

	Switches(offset uint32, limit uint8) ([]Switch, error)
	AddSwitch(dpid uint64, nPorts, firstPort, firstPrintedPort uint16, desc string) (id uint64, duplicated bool, err error)
	RemoveSwitch(id uint64) error

	Networks(offset uint32, limit uint8) ([]Network, error)
	AddNetwork(addr net.IP, mask net.IPMask) (id uint64, duplicated bool, err error)
	RemoveNetwork(id uint64) error
	IPAddrs(networkID uint64) ([]IP, error)
}

func (r *API) Serve() error {
	if r.DB == nil {
		return errors.New("nil DB")
	}
	r.session = newSession(256, 2*time.Hour)

	return r.Server.Serve(
		rest.Post("/api/v1/user/login", r.login),
		rest.Post("/api/v1/user/logout", r.logout),
		rest.Post("/api/v1/user/list", r.listUser),
		rest.Post("/api/v1/user/add", r.addUser),
		rest.Post("/api/v1/user/update", r.updateUser),
		rest.Post("/api/v1/user/activate", r.activateUser),
		rest.Post("/api/v1/user/deactivate", r.deactivateUser),
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
	)
}

func (r *API) validateAdminSession(sessionID string) bool {
	session, ok := r.session.Get(sessionID)
	if ok == false {
		return false
	}

	return session.(*User).Admin
}
