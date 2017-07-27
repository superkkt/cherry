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
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/superkkt/cherry/network"
	"github.com/superkkt/cherry/northbound/app/proxyarp"

	"github.com/dlintw/goconf"
	"github.com/go-sql-driver/mysql"
)

const (
	deadlockErrCode   uint16 = 1213
	foreignkeyErrCode uint16 = 1451
	maxDeadlockRetry         = 5
)

type MySQL struct {
	db     *sql.DB
	random *rand.Rand
}

type config struct {
	host     string
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
		host:     host,
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

	db, err := newDBConn(c.host, c.username, c.password, c.dbName, c.port)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(32)
	db.SetMaxIdleConns(4)

	return &MySQL{
		db:     db,
		random: rand.New(&randomSource{src: rand.NewSource(time.Now().Unix())}),
	}, nil
}

func newDBConn(host, username, password, dbname string, port uint16) (*sql.DB, error) {
	db, err := sql.Open("mysql", fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?timeout=5s&wait_timeout=120", username, password, host, port, dbname))
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

func isForeignkeyErr(err error) bool {
	e, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}

	return e.Number == foreignkeyErrCode
}

func (r *MySQL) query(f func(*sql.DB) error) error {
	deadlockRetry := 0

	for {
		err := f(r.db)
		if err == nil {
			// Success
			return nil
		}
		if !isDeadlock(err) || deadlockRetry >= maxDeadlockRetry {
			return err
		}
		time.Sleep(time.Duration(r.random.Int31n(500)) * time.Millisecond)
		deadlockRetry++
	}
}

func (r *MySQL) MAC(ip net.IP) (mac net.HardwareAddr, ok bool, err error) {
	if ip == nil {
		panic("IP address is nil")
	}

	f := func(db *sql.DB) error {
		// Union query from both vip and host tables
		qry := `(SELECT B.mac 
			 FROM vip 
			 A JOIN host B ON A.active_host_id = B.id 
			 JOIN ip C ON A.ip_id = C.id 
			 WHERE C.address = INET_ATON(?)
		 	) 
			UNION ALL 
			(SELECT mac 
			 FROM host A 
			 JOIN ip B ON A.ip_id = B.id 
			 WHERE B.address = INET_ATON(?)
		 	) 
			LIMIT 1`
		row, err := db.Query(qry, ip.String(), ip.String())
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
			return errors.New("invalid MAC address")
		}
		mac = net.HardwareAddr(v)
		ok = true

		return nil
	}
	err = r.query(f)

	return mac, ok, err
}

func (r *MySQL) Location(mac net.HardwareAddr) (dpid string, port uint32, status network.LocationStatus, err error) {
	if mac == nil {
		panic("MAC address is nil")
	}

	f := func(db *sql.DB) error {
		// Initial value.
		status = network.LocationUnregistered

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		var portID sql.NullInt64
		qry := "SELECT `port_id` FROM `host` WHERE `mac` = ? LOCK IN SHARE MODE"
		if err := tx.QueryRow(qry, []byte(mac)).Scan(&portID); err != nil {
			// Unregistered host?
			if err == sql.ErrNoRows {
				return nil
			} else {
				return err
			}
		}
		// NULL port ID?
		if portID.Valid == false {
			// The node is registered, but we don't know its physical location yet.
			status = network.LocationUndiscovered
			return nil
		}

		qry = "SELECT B.`dpid`, A.`number` FROM `port` A JOIN `switch` B ON A.`switch_id` = B.`id` WHERE A.`id` = ?"
		if err := tx.QueryRow(qry, portID.Int64).Scan(&dpid, &port); err != nil {
			if err == sql.ErrNoRows { // FIXME: Is this possible?
				return nil
			} else {
				return err
			}
		}
		status = network.LocationDiscovered

		return tx.Commit()
	}
	if err := r.query(f); err != nil {
		return "", 0, network.LocationUnregistered, err
	}

	return dpid, port, status, nil
}

func (r *MySQL) Switches() (sw []network.Switch, err error) {
	f := func(db *sql.DB) error {
		rows, err := db.Query("SELECT id, dpid, n_ports, first_port, description FROM switch ORDER BY id DESC")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			v := network.Switch{}
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

func (r *MySQL) AddSwitch(sw network.SwitchParam) (swID uint64, err error) {
	f := func(db *sql.DB) error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		swID, err = r.addSwitch(tx, sw)
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
	if err = r.query(f); err != nil {
		return 0, err
	}

	return swID, nil
}

func (r *MySQL) addSwitch(tx *sql.Tx, sw network.SwitchParam) (swID uint64, err error) {
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

func (r *MySQL) Switch(dpid uint64) (sw network.Switch, ok bool, err error) {
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
		return network.Switch{}, false, err
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
		if isForeignkeyErr(err) {
			return false, errors.New("failed to remove a switch: it has child hosts connected to this switch")
		}
		return false, err
	}

	return ok, nil
}

func (r *MySQL) SwitchPorts(swID uint64) (ports []network.SwitchPort, err error) {
	f := func(db *sql.DB) error {
		qry := `SELECT A.id, A.number, B.first_port
			FROM port A
			JOIN switch B ON A.switch_id = B.id
			WHERE A.switch_id = ?
			ORDER BY id ASC`
		rows, err := db.Query(qry, swID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var firstPort uint
			v := network.SwitchPort{}
			if err := rows.Scan(&v.ID, &v.Number, &firstPort); err != nil {
				return err
			}
			v.Number = v.Number - firstPort + 1
			ports = append(ports, v)
		}

		return rows.Err()
	}
	if err = r.query(f); err != nil {
		return nil, err
	}

	return ports, nil
}

func (r *MySQL) Networks() (networks []network.Network, err error) {
	f := func(db *sql.DB) error {
		qry := `SELECT id, INET_NTOA(address), mask
			FROM network
			ORDER BY id DESC`
		rows, err := db.Query(qry)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			v := network.Network{}
			if err := rows.Scan(&v.ID, &v.Address, &v.Mask); err != nil {
				return err
			}
			networks = append(networks, v)
		}

		return rows.Err()
	}
	if err = r.query(f); err != nil {
		return nil, err
	}

	return networks, nil
}

func (r *MySQL) AddNetwork(addr net.IP, mask net.IPMask) (netID uint64, err error) {
	f := func(db *sql.DB) error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		netID, err = r.addNetwork(tx, addr, mask)
		if err != nil {
			return err
		}
		if err := r.addIPAddrs(tx, netID, addr, mask); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}

		return nil
	}
	if err = r.query(f); err != nil {
		return 0, err
	}

	return netID, nil
}

func (r *MySQL) addNetwork(tx *sql.Tx, addr net.IP, mask net.IPMask) (netID uint64, err error) {
	qry := "INSERT INTO network (address, mask) VALUES (INET_ATON(?), ?)"
	ones, _ := mask.Size()
	result, err := tx.Exec(qry, addr.String(), ones)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint64(id), nil
}

func (r *MySQL) addIPAddrs(tx *sql.Tx, netID uint64, addr net.IP, mask net.IPMask) error {
	stmt, err := tx.Prepare("INSERT INTO ip (network_id, address) VALUES (?, INET_ATON(?) + ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	ones, bits := mask.Size()
	n_addrs := int(math.Pow(2, float64(bits-ones))) - 2 // Minus two due to network and broadcast addresses
	for i := 0; i < n_addrs; i++ {
		if _, err := stmt.Exec(netID, addr.String(), i+1); err != nil {
			return err
		}
	}

	return nil
}

func (r *MySQL) Network(addr net.IP) (n network.Network, ok bool, err error) {
	f := func(db *sql.DB) error {
		row, err := db.Query("SELECT id, INET_NTOA(address), mask FROM network WHERE address = INET_ATON(?)", addr.String())
		if err != nil {
			return err
		}
		defer row.Close()

		// Emptry row?
		if !row.Next() {
			return nil
		}
		if err := row.Scan(&n.ID, &n.Address, &n.Mask); err != nil {
			return err
		}
		ok = true

		return nil
	}
	if err = r.query(f); err != nil {
		return network.Network{}, false, err
	}

	return n, ok, nil
}

func (r *MySQL) RemoveNetwork(id uint64) (ok bool, err error) {
	f := func(db *sql.DB) error {
		result, err := db.Exec("DELETE FROM network WHERE id = ?", id)
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
		if isForeignkeyErr(err) {
			return false, errors.New("failed to remove a network: it has child IP addresses that are being used by hosts")
		}
		return false, err
	}

	return ok, nil
}

func (r *MySQL) IPAddrs(networkID uint64) (addresses []network.IP, err error) {
	f := func(db *sql.DB) error {
		qry := `SELECT A.id, INET_NTOA(A.address), A.used, C.description, IFNULL(CONCAT(E.description, '/', D.number), '')  
			FROM ip A 
			JOIN network B ON A.network_id = B.id 
			LEFT JOIN host C ON C.ip_id = A.id 
			LEFT JOIN port D ON D.id = C.port_id 
			LEFT JOIN switch E ON E.id = D.switch_id 
			WHERE A.network_id = ?`
		rows, err := db.Query(qry, networkID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var host, port sql.NullString
			v := network.IP{}
			if err := rows.Scan(&v.ID, &v.Address, &v.Used, &host, &port); err != nil {
				return err
			}
			v.Host = host.String
			v.Port = port.String
			addresses = append(addresses, v)
		}

		return rows.Err()
	}
	if err = r.query(f); err != nil {
		return nil, err
	}

	return addresses, nil
}

func decodeMAC(s string) (net.HardwareAddr, error) {
	v, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if len(v) != 6 {
		return nil, fmt.Errorf("invalid MAC address: %v", v)
	}

	return net.HardwareAddr(v), nil
}

func (r *MySQL) Hosts() (hosts []network.Host, err error) {
	f := func(db *sql.DB) error {
		qry := `SELECT A.id, CONCAT(INET_NTOA(B.address), '/', E.mask), IFNULL(CONCAT(D.description, '/', C.number), ''), HEX(A.mac), A.description 
			FROM host A 
			JOIN ip B ON A.ip_id = B.id 
			LEFT JOIN port C ON A.port_id = C.id 
			LEFT JOIN switch D ON C.switch_id = D.id 
			JOIN network E ON B.network_id = E.id 
			ORDER by A.id DESC`
		rows, err := db.Query(qry)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			v := network.Host{}
			if err := rows.Scan(&v.ID, &v.IP, &v.Port, &v.MAC, &v.Description); err != nil {
				return err
			}
			mac, err := decodeMAC(v.MAC)
			if err != nil {
				return err
			}
			v.MAC = mac.String()
			hosts = append(hosts, v)
		}

		return rows.Err()
	}
	if err = r.query(f); err != nil {
		return nil, err
	}

	return hosts, nil
}

func (r *MySQL) Host(id uint64) (host network.Host, ok bool, err error) {
	f := func(db *sql.DB) error {
		qry := `SELECT A.id, CONCAT(INET_NTOA(B.address), '/', E.mask), IFNULL(CONCAT(D.description, '/', C.number), ''), HEX(A.mac), A.description 
			FROM host A 
			JOIN ip B ON A.ip_id = B.id 
			LEFT JOIN port C ON A.port_id = C.id 
			LEFT JOIN switch D ON C.switch_id = D.id 
			JOIN network E ON B.network_id = E.id 
			WHERE A.id = ?`
		row, err := db.Query(qry, id)
		if err != nil {
			return err
		}
		defer row.Close()

		if !row.Next() {
			return nil
		}
		if err := row.Scan(&host.ID, &host.IP, &host.Port, &host.MAC, &host.Description); err != nil {
			return err
		}
		mac, err := decodeMAC(host.MAC)
		if err != nil {
			return err
		}
		host.MAC = mac.String()
		ok = true

		return nil
	}
	if err = r.query(f); err != nil {
		return network.Host{}, false, err
	}

	return host, ok, nil
}

func (r *MySQL) AddHost(host network.HostParam) (hostID uint64, err error) {
	f := func(db *sql.DB) error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		ok, err := isAvailableIP(tx, host.IPID)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("already used IP address")
		}
		hostID, err = addNewHost(tx, host)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		return nil
	}
	if err = r.query(f); err != nil {
		return 0, err
	}

	return hostID, nil
}

func addNewHost(tx *sql.Tx, host network.HostParam) (uint64, error) {
	qry := "INSERT INTO host (ip_id, mac, description, last_updated_timestamp) VALUES (?, UNHEX(?), ?, NOW())"
	result, err := tx.Exec(qry, host.IPID, normalizeMAC(host.MAC), host.Description)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint64(id), nil
}

func isAvailableIP(tx *sql.Tx, id uint64) (bool, error) {
	row, err := tx.Query("SELECT used FROM ip WHERE id = ? FOR UPDATE", id)
	if err != nil {
		return false, err
	}
	defer row.Close()
	// Emptry row?
	if !row.Next() {
		return false, errors.New("unknown IP address ID")
	}

	var used bool
	if err := row.Scan(&used); err != nil {
		return false, err
	}

	return !used, nil
}

func normalizeMAC(mac string) string {
	// Remove spaces and colons
	return strings.Replace(strings.Replace(mac, ":", "", -1), " ", "", -1)
}

func (r *MySQL) RemoveHost(id uint64) (ok bool, err error) {
	f := func(db *sql.DB) error {
		result, err := db.Exec("DELETE FROM host WHERE id = ?", id)
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
		if isForeignkeyErr(err) {
			return false, errors.New("failed to remove a host: it has child VIP addresses")
		}
		return false, err
	}

	return ok, nil
}

func (r *MySQL) ToggleVIP(id uint64) (ip net.IP, mac net.HardwareAddr, err error) {
	f := func(db *sql.DB) error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		vip, err := getVIP(tx, id)
		if err != nil {
			return err
		}
		ip = vip.address
		if err := swapVIPHosts(tx, *vip); err != nil {
			return err
		}
		// Get standby's MAC address as the standby host will be active soon!
		mac, err = hostMAC(tx, vip.standby)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		return nil
	}
	if err = r.query(f); err != nil {
		return nil, nil, err
	}

	return ip, mac, nil
}

func getVIP(tx *sql.Tx, id uint64) (*vip, error) {
	qry := `SELECT A.id, INET_NTOA(B.address), A.active_host_id, A.standby_host_id 
		FROM vip A 
		JOIN ip B ON A.ip_id = B.id 
		WHERE A.id = ? 
		FOR UPDATE`
	row, err := tx.Query(qry, id)
	if err != nil {
		return nil, err
	}
	defer row.Close()
	// Emptry row?
	if !row.Next() {
		return nil, fmt.Errorf("unknown VIP (ID=%v)", id)
	}

	v := &vip{}
	var address string
	if err := row.Scan(&v.id, &address, &v.active, &v.standby); err != nil {
		return nil, err
	}
	v.address = net.ParseIP(address)
	if v.address == nil {
		return nil, fmt.Errorf("invalid IPv4 address: %v", address)
	}

	return v, nil
}

func (r *MySQL) TogglePortVIP(swDPID uint64, portNum uint16) (result []proxyarp.VIP, err error) {
	f := func(db *sql.DB) error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		portID, err := portID(tx, swDPID, portNum)
		if err != nil {
			return err
		}
		vips, err := getPortVIPs(tx, portID)
		if err != nil {
			return err
		}
		for _, v := range vips {
			if err := swapVIPHosts(tx, v); err != nil {
				return err
			}
			// Get standby's MAC address as the standby host will be active soon!
			mac, err := hostMAC(tx, v.standby)
			if err != nil {
				return err
			}
			result = append(result, proxyarp.VIP{v.address, mac})
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		return nil
	}
	if err = r.query(f); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *MySQL) ToggleDeviceVIP(swDPID uint64) (result []proxyarp.VIP, err error) {
	f := func(db *sql.DB) error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		vips, err := getDeviceVIPs(tx, swDPID)
		if err != nil {
			return err
		}
		for _, v := range vips {
			if err := swapVIPHosts(tx, v); err != nil {
				return err
			}
			// Get standby's MAC address as the standby host will be active soon!
			mac, err := hostMAC(tx, v.standby)
			if err != nil {
				return err
			}
			result = append(result, proxyarp.VIP{v.address, mac})
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		return nil
	}
	if err = r.query(f); err != nil {
		return nil, err
	}

	return result, nil
}

func portID(tx *sql.Tx, swDPID uint64, portNum uint16) (uint64, error) {
	qry := `SELECT A.id 
		FROM port A 
		JOIN switch B ON A.switch_id = B.id 
		WHERE A.number = ? AND B.dpid = ? 
		LOCK IN SHARE MODE`
	row, err := tx.Query(qry, portNum, swDPID)
	if err != nil {
		return 0, err
	}
	defer row.Close()
	// Emptry row?
	if !row.Next() {
		return 0, fmt.Errorf("unknown switch port (DPID=%v, Number=%v)", swDPID, portNum)
	}

	var id uint64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

type vip struct {
	id      uint64
	address net.IP
	active  uint64
	standby uint64
}

func getPortVIPs(tx *sql.Tx, portID uint64) (result []vip, err error) {
	qry := `SELECT A.id, INET_NTOA(C.address), A.active_host_id, A.standby_host_id 
		FROM vip A 
		JOIN host B ON A.active_host_id = B.id 
		JOIN ip C ON A.ip_id = C.id
		WHERE B.port_id = ? 
		FOR UPDATE`
	rows, err := tx.Query(qry, portID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, active, standby uint64
		var address string
		if err := rows.Scan(&id, &address, &active, &standby); err != nil {
			return nil, err
		}
		ip := net.ParseIP(address)
		if ip == nil {
			return nil, fmt.Errorf("invalid IPv4 address: %v", address)
		}
		result = append(result, vip{id, ip, active, standby})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func getDeviceVIPs(tx *sql.Tx, swDPID uint64) (result []vip, err error) {
	qry := `SELECT A.id, INET_NTOA(E.address), A.active_host_id, A.standby_host_id 
		FROM vip A 
		JOIN host B ON A.active_host_id = B.id 
		JOIN port C ON B.port_id = C.id 
		JOIN switch D ON D.id = C.switch_id 
		JOIN ip E ON E.id = A.ip_id 
		WHERE D.dpid = ? 
		FOR UPDATE`
	rows, err := tx.Query(qry, swDPID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, active, standby uint64
		var address string
		if err := rows.Scan(&id, &address, &active, &standby); err != nil {
			return nil, err
		}
		ip := net.ParseIP(address)
		if ip == nil {
			return nil, fmt.Errorf("invalid IPv4 address: %v", address)
		}
		result = append(result, vip{id, ip, active, standby})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func swapVIPHosts(tx *sql.Tx, v vip) error {
	qry := "UPDATE vip SET active_host_id = ?, standby_host_id = ? WHERE id = ?"
	// Swap active and standby hosts
	_, err := tx.Exec(qry, v.standby, v.active, v.id)
	if err != nil {
		return err
	}

	return nil
}

func hostMAC(tx *sql.Tx, hostID uint64) (net.HardwareAddr, error) {
	row, err := tx.Query("SELECT HEX(mac) FROM host WHERE id = ?", hostID)
	if err != nil {
		return nil, err
	}
	defer row.Close()
	// Emptry row?
	if !row.Next() {
		return nil, fmt.Errorf("unknown host (ID=%v)", hostID)
	}

	var v string
	if err := row.Scan(&v); err != nil {
		return nil, err
	}
	mac, err := decodeMAC(v)
	if err != nil {
		return nil, err
	}

	return mac, nil
}

func (r *MySQL) VIPs() (result []network.VIP, err error) {
	vips, err := r.getVIPs()
	if err != nil {
		return nil, err
	}

	for _, v := range vips {
		active, ok, err := r.Host(v.active)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("unknown active host (ID=%v)", v.active)
		}

		standby, ok, err := r.Host(v.standby)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("unknown standby host (ID=%v)", v.standby)
		}

		result = append(result, network.VIP{
			ID:          v.id,
			IP:          v.address,
			ActiveHost:  active,
			StandbyHost: standby,
			Description: v.description,
		})
	}

	return result, nil
}

type registeredVIP struct {
	id          uint64
	address     string
	active      uint64
	standby     uint64
	description string
}

func (r *MySQL) getVIPs() (result []registeredVIP, err error) {
	f := func(db *sql.DB) error {
		qry := `SELECT A.id, CONCAT(INET_NTOA(B.address), '/', C.mask), A.active_host_id, A.standby_host_id, A.description 
			FROM vip A 
			JOIN ip B ON A.ip_id = B.id 
			JOIN network C ON C.id = B.network_id 
			ORDER BY A.id DESC`
		rows, err := db.Query(qry)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var id, active, standby uint64
			var address, description string
			if err := rows.Scan(&id, &address, &active, &standby, &description); err != nil {
				return err
			}
			result = append(result, registeredVIP{id, address, active, standby, description})
		}

		return rows.Err()
	}
	if err = r.query(f); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *MySQL) AddVIP(vip network.VIPParam) (id uint64, cidr string, err error) {
	f := func(db *sql.DB) error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		ok, err := isAvailableIP(tx, vip.IPID)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("already used IP address")
		}
		id, err = addNewVIP(tx, vip)
		if err != nil {
			return err
		}
		cidr, err = getIP(tx, vip.IPID)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		return nil
	}
	if err = r.query(f); err != nil {
		return 0, "", err
	}

	return id, cidr, nil
}

func addNewVIP(tx *sql.Tx, vip network.VIPParam) (uint64, error) {
	qry := "INSERT INTO vip (ip_id, active_host_id, standby_host_id, description) VALUES (?, ?, ?, ?)"
	result, err := tx.Exec(qry, vip.IPID, vip.ActiveHostID, vip.StandbyHostID, vip.Description)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint64(id), nil
}

func getIP(tx *sql.Tx, id uint64) (cidr string, err error) {
	qry := `SELECT CONCAT(INET_NTOA(A.address), '/', B.mask) 
		FROM ip A 
		JOIN network B ON A.network_id = B.id 
		WHERE A.id = ?`
	row, err := tx.Query(qry, id)
	if err != nil {
		return "", err
	}
	defer row.Close()
	// Emptry row?
	if !row.Next() {
		return "", fmt.Errorf("unknown IP (ID=%v)", id)
	}

	if err := row.Scan(&cidr); err != nil {
		return "", err
	}

	return cidr, nil
}

func (r *MySQL) RemoveVIP(id uint64) (ok bool, err error) {
	f := func(db *sql.DB) error {
		result, err := db.Exec("DELETE FROM vip WHERE id = ?", id)
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

// GetUndiscoveredHosts returns IP addresses whose physical location is still
// undiscovered or staled more than expiration. result can be nil on empty result.
func (r *MySQL) GetUndiscoveredHosts(expiration time.Duration) (result []net.IP, err error) {
	f := func(db *sql.DB) error {
		// NOTE: Do not include VIP addresses!
		qry := "SELECT IFNULL(INET_NTOA(B.`address`), '0.0.0.0') "
		qry += "FROM `host` A "
		qry += "JOIN `ip` B "
		qry += "ON A.`ip_id` = B.`id` "
		qry += "WHERE A.`port_id` IS NULL OR A.`last_updated_timestamp` < NOW() - INTERVAL ? SECOND"

		rows, err := db.Query(qry, uint64(expiration.Seconds()))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var addr string
			if err := rows.Scan(&addr); err != nil {
				return err
			}
			ip := net.ParseIP(addr)
			if ip == nil {
				return fmt.Errorf("invalid IP address: %v", addr)
			}
			if ip.IsUnspecified() {
				continue
			}
			result = append(result, ip)
		}

		return rows.Err()
	}

	if err = r.query(f); err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateHostLocation updates the physical location of a host, whose MAC and IP
// addresses are matched with mac and ip, to the port identified by swDPID and
// portNum. updated will be true if its location has been actually updated.
func (r *MySQL) UpdateHostLocation(mac net.HardwareAddr, ip net.IP, swDPID uint64, portNum uint16) (updated bool, err error) {
	f := func(db *sql.DB) error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		hostID, ok, err := getHostID(tx, mac, ip)
		if err != nil {
			return err
		}
		// Unknown host?
		if !ok {
			updated = false
			return nil
		}

		portID, err := portID(tx, swDPID, portNum)
		if err != nil {
			return err
		}

		updated, err = updateLocation(tx, hostID, portID)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		return nil
	}
	if err = r.query(f); err != nil {
		return false, err
	}

	return updated, nil
}

func getHostID(tx *sql.Tx, mac net.HardwareAddr, ip net.IP) (hostID uint64, ok bool, err error) {
	qry := "SELECT A.`id` "
	qry += "FROM `host` A "
	qry += "JOIN `ip` B "
	qry += "ON A.`ip_id` = B.`id` "
	qry += "WHERE A.`mac` = ? AND B.`address` = INET_ATON(?) "
	qry += "LOCK IN SHARE MODE"

	row, err := tx.Query(qry, []byte(mac), ip.String())
	if err != nil {
		return 0, false, err
	}
	defer row.Close()

	// Emptry row?
	if !row.Next() {
		return 0, false, nil
	}
	if err := row.Scan(&hostID); err != nil {
		return 0, false, err
	}

	return hostID, true, nil
}

func updateLocation(tx *sql.Tx, hostID, portID uint64) (updated bool, err error) {
	var id uint64
	qry := "SELECT `id` FROM `host` WHERE `id` = ? AND `port_id` = ?"
	err = tx.QueryRow(qry, hostID, portID).Scan(&id)
	// Real error?
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	// Need to update?
	if err == sql.ErrNoRows {
		updated = true
	}
	qry = "UPDATE `host` SET `port_id` = ?, `last_updated_timestamp` = NOW() WHERE `id` = ?"
	_, err = tx.Exec(qry, portID, hostID)
	if err != nil {
		return false, err
	}

	return updated, nil
}

// ResetHostLocationsByPort sets NULL to the host locations that belong to the
// port specified by swDPID and portNum.
func (r *MySQL) ResetHostLocationsByPort(swDPID uint64, portNum uint16) error {
	f := func(db *sql.DB) error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		portID, err := portID(tx, swDPID, portNum)
		if err != nil {
			return err
		}

		qry := "UPDATE `host` SET `port_id` = NULL WHERE `port_id` = ?"
		if _, err := tx.Exec(qry, portID); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		return nil
	}

	return r.query(f)
}

// ResetHostLocationsByDevice sets NULL to the host locations that belong to the
// device specified by swDPID.
func (r *MySQL) ResetHostLocationsByDevice(swDPID uint64) error {
	f := func(db *sql.DB) error {
		qry := "UPDATE `host` A "
		qry += "JOIN `port` B ON A.`port_id` = B.`id` "
		qry += "JOIN `switch` C ON B.`switch_id` = C.`id` "
		qry += "SET A.`port_id` = NULL "
		qry += "WHERE C.`dpid` = ?"

		_, err := db.Exec(qry, swDPID)
		if err != nil {
			return err
		}

		return nil
	}

	return r.query(f)
}
