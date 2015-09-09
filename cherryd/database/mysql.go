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
	"net"
	"strings"

	"github.com/dlintw/goconf"
	"github.com/go-sql-driver/mysql"
	"github.com/superkkt/cherry/cherryd/network"
)

const (
	deadlockErrCode  uint16 = 1213
	maxDeadlockRetry        = 5
)

type MySQL struct {
	db []*sql.DB
}

type config struct {
	hosts    []string
	port     uint16
	username string
	password string
	dbName   string
}

func parseConfig(conf *goconf.ConfigFile) (*config, error) {
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

	v := &config{
		hosts:    strings.Split(strings.Replace(host, " ", "", -1), ","),
		port:     uint16(port),
		username: user,
		password: password,
		dbName:   dbname,
	}
	return v, nil
}

func NewMySQL(conf *goconf.ConfigFile) (*MySQL, error) {
	c, err := parseConfig(conf)
	if err != nil {
		return nil, err
	}

	db := make([]*sql.DB, 0)
	var lastErr error
	for _, host := range c.hosts {
		v, err := newDBConn(host, c.username, c.password, c.dbName, c.port)
		if err != nil {
			lastErr = err
			continue
		}
		v.SetMaxOpenConns(32)
		v.SetMaxIdleConns(4)
		db = append(db, v)
	}
	if len(db) == 0 {
		return nil, fmt.Errorf("no avaliable database server: %v", lastErr)
	}
	mysql := &MySQL{
		db: db,
	}

	return mysql, nil
}

func newDBConn(host, username, password, dbname string, port uint16) (*sql.DB, error) {
	db, err := sql.Open("mysql", fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?timeout=5s", username, password, host, port, dbname))
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func isDeadlock(err error) bool {
	e, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}

	return e.Number == deadlockErrCode
}

func isConnectionError(err error) bool {
	e, ok := err.(*mysql.MySQLError)
	// Assume all errors except MySQLError are connection failure
	if !ok || e.Number >= 2000 {
		return true
	}

	return false
}

func (r *MySQL) query(f func(*sql.DB) error) error {
	var err error

	for _, db := range r.db {
		deadlockRetry := 0

	retry:
		err = f(db)
		if err == nil {
			return nil
		}
		if isConnectionError(err) {
			// Use other DB server if we got connection failure
			continue
		}

		if !isDeadlock(err) || deadlockRetry >= maxDeadlockRetry {
			return err
		}
		deadlockRetry++
		goto retry
	}

	return err
}

func (r *MySQL) MAC(ip net.IP) (mac net.HardwareAddr, ok bool, err error) {
	if ip == nil {
		panic("IP address is nil")
	}

	f := func(db *sql.DB) error {
		qry := `SELECT mac 
			FROM host A 
			JOIN ip B 
			ON A.ip_id = B.id 
			WHERE B.address = INET_ATON(?)`
		row, err := db.Query(qry, ip.String())
		if err != nil {
			return err
		}
		defer row.Close()

		// Unknown IP address?
		if !row.Next() {
			return nil
		}
		if err := row.Err(); err != nil {
			return err
		}

		var v []byte
		if err := row.Scan(&v); err != nil {
			return err
		}
		if v == nil || len(v) != 6 {
			panic("Invalid MAC address")
		}
		mac = net.HardwareAddr(v)
		ok = true

		return nil
	}
	err = r.query(f)

	return mac, ok, err
}

func (r *MySQL) Location(mac net.HardwareAddr) (dpid string, port uint32, ok bool, err error) {
	if mac == nil {
		panic("MAC address is nil")
	}

	f := func(db *sql.DB) error {
		qry := `SELECT A.dpid, B.number 
			FROM switch A 
			JOIN port B 
			ON B.switch_id = A.id 
			JOIN host C 
			ON C.port_id = B.id 
			WHERE C.mac = ?
			GROUP BY(A.dpid)`
		row, err := db.Query(qry, []byte(mac))
		if err != nil {
			return err
		}
		defer row.Close()

		// Unknown MAC address?
		if !row.Next() {
			return nil
		}
		if err := row.Err(); err != nil {
			return err
		}

		if err := row.Scan(&dpid, &port); err != nil {
			return err
		}
		ok = true

		return nil
	}
	err = r.query(f)

	return dpid, port, ok, err
}

func (r *MySQL) Switches() (sw []network.RegisteredSwitch, err error) {
	f := func(db *sql.DB) error {
		rows, err := db.Query("SELECT id, dpid, n_ports, first_port, description FROM switch ORDER BY id DESC")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			v := network.RegisteredSwitch{}
			if err := rows.Scan(&v.ID, &v.DPID, &v.NumPorts, &v.FirstPort, &v.Description); err != nil {
				return err
			}
			sw = append(sw, v)
		}

		return rows.Err()
	}
	if err = r.query(f); err != nil {
		return nil, err
	}

	return sw, nil
}

func (r *MySQL) AddSwitch(sw network.Switch) error {
	f := func(db *sql.DB) error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		swID, err := r.addSwitch(tx, sw)
		if err != nil {
			return err
		}
		if err := r.addPorts(tx, swID, sw.FirstPort, sw.NumPorts); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}

		return nil
	}

	return r.query(f)
}

func (r *MySQL) addSwitch(tx *sql.Tx, sw network.Switch) (swID uint64, err error) {
	qry := "INSERT INTO switch (dpid, n_ports, first_port, description) VALUES (?, ?, ?, ?)"
	result, err := tx.Exec(qry, sw.DPID, sw.NumPorts, sw.FirstPort, sw.Description)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint64(id), nil
}

func (r *MySQL) addPorts(tx *sql.Tx, swID uint64, firstPort, n_ports uint16) error {
	stmt, err := tx.Prepare("INSERT INTO port (switch_id, number) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := uint16(0); i < n_ports; i++ {
		if _, err := stmt.Exec(swID, firstPort+i); err != nil {
			return err
		}
	}

	return nil
}

func (r *MySQL) Switch(dpid uint64) (sw network.RegisteredSwitch, ok bool, err error) {
	f := func(db *sql.DB) error {
		row, err := db.Query("SELECT id, dpid, n_ports, first_port, description FROM switch WHERE dpid = ?", dpid)
		if err != nil {
			return err
		}
		defer row.Close()

		// Emptry row?
		if !row.Next() {
			return nil
		}
		if err := row.Scan(&sw.ID, &sw.DPID, &sw.NumPorts, &sw.FirstPort, &sw.Description); err != nil {
			return err
		}
		ok = true

		return nil
	}
	if err = r.query(f); err != nil {
		return network.RegisteredSwitch{}, false, err
	}

	return sw, ok, nil
}

func (r *MySQL) RemoveSwitch(id uint64) (ok bool, err error) {
	f := func(db *sql.DB) error {
		result, err := db.Exec("DELETE FROM switch WHERE id = ?", id)
		if err != nil {
			return err
		}
		nRows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if nRows > 0 {
			ok = true
		}

		return nil
	}
	if err = r.query(f); err != nil {
		return false, err
	}

	return ok, nil
}
