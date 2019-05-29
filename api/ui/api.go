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
	"fmt"
	"net"
	"strconv"
	"strings"
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
	LogTransaction
	CategoryTransaction
}

type Search struct {
	Key   Column `json:"key"`
	Value string `json:"value"`
}

func (r *Search) Validate() error {
	switch r.Key {
	case ColumnIP:
		return validateIP(r.Value)
	case ColumnMAC:
		return validateMAC(r.Value)
	case ColumnLogType:
		return LogType(r.Value).Validate()
	case ColumnLogMethod:
		return LogMethod(r.Value).Validate()
	case ColumnPort, ColumnGroup, ColumnDescription, ColumnUser:
		if len(r.Value) == 0 {
			return errors.New("empty search value")
		}
		return nil
	default:
		return fmt.Errorf("invalid search key: %v", r.Key)
	}
}

// IP format is '1.*.*.*', '1.2.*.*', '1.2.3.*', '1.2.3.4'.
func validateIP(ip string) error {
	invalid := fmt.Errorf("invalid IP address: %v", ip)

	token := strings.Split(ip, ".")
	if len(token) != 4 {
		return invalid
	}

	var wildcard [4]bool
	for i, v := range token {
		if v == "*" {
			wildcard[i] = true
			continue
		}
		d, err := strconv.Atoi(v)
		if err != nil || (d < 0 || d > 255) {
			return invalid
		}
	}

	if wildcard[0] == true {
		return invalid
	}
	if wildcard[1] == true && (wildcard[2] == false || wildcard[3] == false) {
		return invalid
	}
	if wildcard[2] == true && wildcard[3] == false {
		return invalid
	}

	return nil
}

// MAC format is 'A1:*:*:*:*:*', 'A1:A2:*:*:*:*', 'A1:A2:A3:*:*:*', 'A1:A2:A3:A4:*:*', 'A1:A2:A3:A4:A5:*', 'A1:A2:A3:A4:A5:A6'.
func validateMAC(mac string) error {
	invalid := fmt.Errorf("invalid MAC address: %v", mac)

	token := strings.Split(mac, ":")
	if len(token) != 6 {
		return invalid
	}

	var wildcard [6]bool
	for i, v := range token {
		if v == "*" {
			wildcard[i] = true
			continue
		}
		d, err := strconv.ParseUint(v, 16, 8)
		if len(v) != 2 || err != nil || (d < 0 || d > 255) {
			return invalid
		}
	}

	if wildcard[0] == true {
		return invalid
	}
	if wildcard[1] == true && (wildcard[2] == false || wildcard[3] == false || wildcard[4] == false || wildcard[5] == false) {
		return invalid
	}
	if wildcard[2] == true && (wildcard[3] == false || wildcard[4] == false || wildcard[5] == false) {
		return invalid
	}
	if wildcard[3] == true && (wildcard[4] == false || wildcard[5] == false) {
		return invalid
	}
	if wildcard[4] == true && wildcard[5] == false {
		return invalid
	}

	return nil
}

type Sort struct {
	Key   Column `json:"key"`
	Order Order  `json:"order"`
}

func (r *Sort) Validate() error {
	if r.Order <= OrderInvalid || r.Order > OrderDescending {
		return errors.New("invalid sort order")
	}
	if r.Key <= ColumnInvalid || r.Key > ColumnGroup {
		return fmt.Errorf("invalid sort key: %v", r.Key)
	}

	return nil
}

type Column int

const (
	ColumnInvalid Column = iota
	ColumnTime
	ColumnIP
	ColumnMAC
	ColumnPort
	ColumnGroup
	ColumnDescription
	ColumnUser
	ColumnLogType
	ColumnLogMethod
)

type Order int

const (
	OrderInvalid Order = iota
	OrderAscending
	OrderDescending
)

type Pagination struct {
	Offset uint32 `json:"offset"`
	Limit  uint8  `json:"limit"`
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
		rest.Post("/api/v1/group/list", api.ResponseHandler(r.listGroup)),
		rest.Post("/api/v1/group/add", api.ResponseHandler(r.addGroup)),
		rest.Post("/api/v1/group/update", api.ResponseHandler(r.updateGroup)),
		rest.Post("/api/v1/group/remove", api.ResponseHandler(r.removeGroup)),
		rest.Post("/api/v1/switch/list", api.ResponseHandler(r.listSwitch)),
		rest.Post("/api/v1/switch/add", api.ResponseHandler(r.addSwitch)),
		rest.Post("/api/v1/switch/remove", api.ResponseHandler(r.removeSwitch)),
		rest.Post("/api/v1/network/list", api.ResponseHandler(r.listNetwork)),
		rest.Post("/api/v1/network/add", api.ResponseHandler(r.addNetwork)),
		rest.Post("/api/v1/network/remove", api.ResponseHandler(r.removeNetwork)),
		rest.Post("/api/v1/network/ip", api.ResponseHandler(r.listIP)),
		rest.Post("/api/v1/host/list", api.ResponseHandler(r.listHost)),
		rest.Post("/api/v1/host/add", api.ResponseHandler(r.addHost)),
		rest.Post("/api/v1/host/update", api.ResponseHandler(r.updateHost)),
		rest.Post("/api/v1/host/activate", api.ResponseHandler(r.activateHost)),
		rest.Post("/api/v1/host/deactivate", api.ResponseHandler(r.deactivateHost)),
		rest.Post("/api/v1/host/remove", api.ResponseHandler(r.removeHost)),
		rest.Post("/api/v1/vip/list", api.ResponseHandler(r.listVIP)),
		rest.Post("/api/v1/vip/add", api.ResponseHandler(r.addVIP)),
		rest.Post("/api/v1/vip/remove", api.ResponseHandler(r.removeVIP)),
		rest.Post("/api/v1/vip/toggle", api.ResponseHandler(r.toggleVIP)),
		rest.Post("/api/v1/log/list", api.ResponseHandler(r.listLog)),
		rest.Post("/api/v1/category/list", api.ResponseHandler(r.listCategory)),
		rest.Post("/api/v1/category/add", api.ResponseHandler(r.addCategory)),
		rest.Post("/api/v1/category/update", api.ResponseHandler(r.updateCategory)),
		rest.Post("/api/v1/category/remove", api.ResponseHandler(r.removeCategory)),
	)
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
