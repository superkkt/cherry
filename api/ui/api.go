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
	"net/http"
	"time"

	"github.com/superkkt/cherry/api"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/davecgh/go-spew/spew"
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

	Host(id uint64) (*Host, error)
	AddHost(ipID []uint64, groupID *uint64, mac net.HardwareAddr, desc string) (host []*Host, duplicated bool, err error)
	UpdateHost(id, ipID uint64, groupID *uint64, mac net.HardwareAddr, desc string) (host *Host, duplicated bool, err error)
	// ActivateHost enables a host specified by id and then returns information of the host. It returns nil if the host does not exist.
	ActivateHost(id uint64) (*Host, error)
	// DeactivateHost disables a host specified by id and then returns information of the host. It returns nil if the host does not exist.
	DeactivateHost(id uint64) (*Host, error)
	// RemoveHost removes a host specified by id and then returns information of the host before removing. It returns nil if the host does not exist.
	RemoveHost(id uint64) (*Host, error)

	VIP(id uint64) (*VIP, error)
	VIPs(offset uint32, limit uint8) ([]VIP, error)
	AddVIP(ipID, activeID, standbyID uint64, desc string) (id uint64, duplicated bool, err error)
	RemoveVIP(id uint64) error
	ToggleVIP(id uint64) error
}

func (r *API) Serve() error {
	if r.DB == nil {
		return errors.New("nil DB")
	}
	r.session = newSession(256, 2*time.Hour)

	return r.Server.Serve(
		rest.Post("/api/v1/user/login", handler(r.login)),
		rest.Post("/api/v1/user/logout", handler(r.logout)),
		rest.Post("/api/v1/user/list", handler(r.listUser)),
		rest.Post("/api/v1/user/add", handler(r.addUser)),
		rest.Post("/api/v1/user/update", handler(r.updateUser)),
		rest.Post("/api/v1/user/activate", handler(r.activateUser)),
		rest.Post("/api/v1/user/deactivate", handler(r.deactivateUser)),
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

func handler(f func(responseWriter, *rest.Request)) func(rest.ResponseWriter, *rest.Request) {
	return func(w rest.ResponseWriter, req *rest.Request) {
		lw := &logWriter{w: w}
		f(lw, req)
	}
}

type responseWriter interface {
	// Identical to the http.ResponseWriter interface
	Header() http.Header

	Write(api.Response)

	// Similar to the http.ResponseWriter interface, with additional JSON related
	// headers set.
	WriteHeader(int)
}

type logWriter struct {
	w rest.ResponseWriter
}

func (r *logWriter) Header() http.Header {
	return r.w.Header()
}

func (r *logWriter) Write(resp api.Response) {
	switch {
	case resp.Status >= api.StatusInternalServerError:
		logger.Errorf("server-side error response: status=%v, message=%v", resp.Status, resp.Message)
	case resp.Status >= api.StatusInvalidParameter:
		logger.Infof("client-side error response: status=%v, message=%v", resp.Status, resp.Message)
	default:
		logger.Debugf("success response: %v", spew.Sdump(resp))
	}

	if err := r.w.WriteJson(resp); err != nil {
		logger.Errorf("failed to write a JSON response: %v", err)
	}
}

func (r *logWriter) WriteHeader(status int) {
	r.w.WriteHeader(status)
}
