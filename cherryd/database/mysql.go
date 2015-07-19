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
	"database/sql"
	"errors"
	"fmt"
	"github.com/dlintw/goconf"
	_ "github.com/go-sql-driver/mysql"
	"net"
)

type MySQL struct {
	db *sql.DB
}

func NewMySQL(conf *goconf.ConfigFile) (*MySQL, error) {
	db, err := newDBConn(conf)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(64)
	db.SetMaxIdleConns(8)

	mysql := &MySQL{
		db: db,
	}
	if err := mysql.createTables(); err != nil {
		return nil, err
	}

	return mysql, nil
}

func newDBConn(conf *goconf.ConfigFile) (*sql.DB, error) {
	host, err := conf.GetString("database", "host")
	if err != nil || len(host) == 0 {
		return nil, errors.New("empty database host in the config file")
	}
	port, err := conf.GetInt("database", "port")
	if err != nil || port <= 0 || port > 0xFFFF {
		return nil, errors.New("invalid database port in the config file")
	}
	user, err := conf.GetString("database", "user")
	if err != nil || len(user) == 0 {
		return nil, errors.New("empty database user in the config file")
	}
	password, err := conf.GetString("database", "password")
	if err != nil || len(password) == 0 {
		return nil, errors.New("empty database password in the config file")
	}
	dbname, err := conf.GetString("database", "name")
	if err != nil || len(dbname) == 0 {
		return nil, errors.New("empty database name in the config file")
	}

	db, err := sql.Open("mysql", fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?timeout=5s", user, password, host, port, dbname))
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func (r *MySQL) FindMAC(ip net.IP) (mac net.HardwareAddr, ok bool, err error) {
	if ip == nil {
		panic("IP address is nil")
	}

	qry := `SELECT mac 
		FROM host A 
		JOIN ip B 
		ON A.ip_id = B.id 
		WHERE B.address = INET_ATON(?)`
	row, err := r.db.Query(qry, ip.String())
	if err != nil {
		return nil, false, err
	}
	defer row.Close()

	// Unknown IP address?
	if !row.Next() {
		return nil, false, nil
	}
	if err := row.Err(); err != nil {
		return nil, false, err
	}

	var v []byte
	if err := row.Scan(&v); err != nil {
		return nil, false, err
	}
	if v == nil || len(v) != 6 {
		panic("Invalid MAC address")
	}

	return net.HardwareAddr(v), true, nil
}

func (r *MySQL) GetNetworks() ([]*net.IPNet, error) {
	qry := `SELECT INET_NTOA(address), mask 
		FROM network`
	row, err := r.db.Query(qry)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	result := make([]*net.IPNet, 0)
	for row.Next() {
		var addr, mask string
		if err := row.Scan(&addr, &mask); err != nil {
			return nil, err
		}
		_, ipnet, err := net.ParseCIDR(fmt.Sprintf("%v/%v", addr, mask))
		if err != nil {
			return nil, err
		}
		result = append(result, ipnet)
	}
	if err := row.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
