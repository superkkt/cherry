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
	"time"

	"gopkg.in/ldap.v3"
)

func (r *Client) acquireConn() (conn *ldap.Conn, err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.numConn == 0 {
		r.cond.Wait()
	}

	return r.createConn()
}

func (r *Client) createConn() (*ldap.Conn, error) {
	conn, err := ldap.DialTLS("tcp", r.config.GetString("addr"), r.tls)
	if err != nil {
		return nil, err
	}

	conn.SetTimeout(30 * time.Second)

	r.numConn--
	if r.numConn < 0 {
		panic("negative numConn value")
	}

	return conn, nil
}

func (r *Client) releaseConn(conn *ldap.Conn) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	conn.Close()
	r.numConn++
	r.cond.Signal()
}
