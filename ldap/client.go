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

package ldap

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/superkkt/viper"
	"gopkg.in/ldap.v3"
)

type Client struct {
	mutex *sync.Mutex
	cond  *sync.Cond

	config  *viper.Viper
	tls     *tls.Config
	numConn int
}

func New(config *viper.Viper, maxConn int) *Client {
	mutex := new(sync.Mutex)

	return &Client{
		mutex:   mutex,
		cond:    sync.NewCond(mutex),
		config:  config,
		tls:     &tls.Config{InsecureSkipVerify: true},
		numConn: maxConn,
	}
}

func (r *Client) Auth(username, password string) (ok bool, err error) {
	conn, err := r.acquireConn()
	if err != nil {
		return false, err
	}
	defer r.releaseConn(conn)

	if err := r.bindAdmin(conn); err != nil {
		return false, err
	}

	dn, err := r.getDN(conn, username)
	if err != nil {
		return false, err
	}
	// Incorrect username.
	if len(dn) == 0 {
		return false, nil
	}

	if err = conn.Bind(dn, password); err != nil {
		if e, ok := err.(*ldap.Error); ok {
			// Incorrect password.
			if e.ResultCode == ldap.LDAPResultInvalidCredentials {
				return false, nil
			}
		}
		return false, err
	}

	return true, nil
}

func (r *Client) bindAdmin(conn *ldap.Conn) error {
	return conn.Bind(r.config.GetString("admin.name"), r.config.GetString("admin.password"))
}

// It returns empty string, if no user matches username.
func (r *Client) getDN(conn *ldap.Conn, username string) (dn string, err error) {
	result, err := conn.Search(ldap.NewSearchRequest(
		r.config.GetString("base_dn"),
		ldap.ScopeWholeSubtree, ldap.DerefAlways, 1, 0, false,
		fmt.Sprintf("(&(%v=%v)(objectclass=user))", r.config.GetString("attr.login"), username),
		[]string{"DN"},
		nil,
	))
	if err != nil {
		return "", err
	}
	if len(result.Entries) == 0 {
		return "", nil
	}

	return result.Entries[0].DN, nil
}
