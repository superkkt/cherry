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
	"runtime"
	"strings"
	"time"

	"github.com/superkkt/cherry/api"
	"github.com/superkkt/cherry/network"
	"github.com/superkkt/cherry/northbound/app/announcer"
	"github.com/superkkt/cherry/northbound/app/discovery"
	"github.com/superkkt/cherry/northbound/app/virtualip"

	"github.com/go-sql-driver/mysql"
	"github.com/superkkt/go-logging"
	"github.com/superkkt/viper"
)

const (
	maxDeadlockRetry = 5

	deadlockErrCode   uint16 = 1213
	duplicatedErrCode uint16 = 1062
	foreignkeyErrCode uint16 = 1451

	clusterDialerNetwork = "cluster"
)

var (
	logger = logging.MustGetLogger("database")

	maxIdleConn = runtime.NumCPU()
	maxOpenConn = maxIdleConn * 2
)

type MySQL struct {
	db     *sql.DB
	random *rand.Rand
}

func NewMySQL() (*MySQL, error) {
	addr := viper.GetString("mysql.addr")
	if err := validateClusterAddr(addr); err != nil {
		return nil, err
	}
	// Register the custom dialer.
	mysql.RegisterDial(clusterDialerNetwork, clusterDialer)

	param := "readTimeout=1m&writeTimeout=1m&parseTime=true&loc=Local&maxAllowedPacket=0"
	dsn := fmt.Sprintf("%v:%v@%v(%v)/%v?%v", viper.GetString("mysql.username"), viper.GetString("mysql.password"), clusterDialerNetwork, addr, viper.GetString("mysql.name"), param)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(maxOpenConn)
	db.SetMaxIdleConns(maxIdleConn)
	// Make sure that all the connections are established to a same node, instead of distributing them into multiple nodes.
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &MySQL{
		db:     db,
		random: rand.New(&randomSource{src: rand.NewSource(time.Now().Unix())}),
	}, nil
}

func validateClusterAddr(addr string) error {
	if len(addr) == 0 {
		return errors.New("empty cluster address")
	}

	token := strings.Split(strings.Replace(addr, " ", "", -1), ",")
	if len(token) == 0 {
		return fmt.Errorf("invalid cluster address: %v", addr)
	}

	for _, v := range token {
		if _, err := net.ResolveTCPAddr("tcp", v); err != nil {
			return fmt.Errorf("invalid cluster address: %v: %v", v, err)
		}
	}

	return nil
}

// clusterDialer tries to sequentially connect to each hosts from the address in the
// order of their appearance and then returns the first successfully connected one.
func clusterDialer(addr string) (net.Conn, error) {
	token := strings.Split(strings.Replace(addr, " ", "", -1), ",")

	for _, v := range token {
		logger.Debugf("dialing to %v", v)
		conn, err := net.DialTimeout("tcp", v, 5*time.Second)
		if err == nil {
			// Connected!
			logger.Debugf("successfully connected to %v", v)
			return conn, nil
		}
		logger.Errorf("failed to dial: %v", err)
	}

	return nil, errors.New("failed to dial: no available cluster node")
}

func isDeadlock(err error) bool {
	e, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}

	return e.Number == deadlockErrCode
}

func isDuplicated(err error) bool {
	e, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}

	return e.Number == duplicatedErrCode
}

func isForeignkeyErr(err error) bool {
	e, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}

	return e.Number == foreignkeyErrCode
}

func (r *MySQL) query(f func(*sql.Tx) error) error {
	deadlockRetry := 0

	for {
		tx, err := r.db.Begin()
		if err != nil {
			return err
		}

		err = f(tx)
		// Success?
		if err == nil {
			// Yes! but Commit also may raise an error.
			err = tx.Commit()
			// Success?
			if err == nil {
				// Transaction committed successfully!
				return nil
			}
			// Fallthrough!
		}
		// No! query failed.
		tx.Rollback()

		// Need to retry due to a deadlock?
		if !isDeadlock(err) || deadlockRetry >= maxDeadlockRetry {
			// No, do not retry and just return the error.
			return err
		}
		// Yes, a deadlock occurrs. Re-execute the queries again after some sleep!
		logger.Infof("query failed due to a deadlock: caller=%v", caller())
		time.Sleep(time.Duration(rand.Int31n(500)) * time.Millisecond)
		deadlockRetry++
	}
}

func caller() string {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}

	f := runtime.FuncForPC(pc)
	if f == nil {
		return fmt.Sprintf("%v:%v", file, line)
	}

	return fmt.Sprintf("%v (%v:%v)", f.Name(), file, line)
}

func (r *MySQL) Auth(name, password string) (user *api.User, err error) {
	f := func(tx *sql.Tx) error {
		v := new(api.User)
		qry := "SELECT `id`, `name`, `enabled`, `admin`, `timestamp` FROM `user` WHERE `name` = ? AND `password` = SHA2(?, 256)"
		if err := tx.QueryRow(qry, name, name+password).Scan(&v.ID, &v.Name, &v.Enabled, &v.Admin, &v.Timestamp); err != nil {
			if err == sql.ErrNoRows {
				// Incorrect credential or not exist.
				return nil
			}
			return err
		}
		user = v

		return nil
	}
	if err = r.query(f); err != nil {
		return nil, err
	}

	return user, nil
}

func (r *MySQL) Users(offset uint32, limit uint8) (user []api.User, err error) {
	f := func(tx *sql.Tx) error {
		qry := "SELECT `id`, `name`, `enabled`, `admin`, `timestamp` "
		qry += "FROM `user` "
		qry += "ORDER BY `id` DESC "
		qry += "LIMIT ?, ?"

		rows, err := tx.Query(qry, offset, limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		user = []api.User{}
		for rows.Next() {
			v := api.User{}
			if err := rows.Scan(&v.ID, &v.Name, &v.Enabled, &v.Admin, &v.Timestamp); err != nil {
				return err
			}
			user = append(user, v)
		}

		return rows.Err()
	}

	if err = r.query(f); err != nil {
		return nil, err
	}

	return user, nil
}

func (r *MySQL) AddUser(name, password string) (userID uint64, duplicated bool, err error) {
	f := func(tx *sql.Tx) error {
		qry := "INSERT INTO `user` (`name`, `password`, `enabled`, `admin`, `timestamp`) "
		qry += "VALUES (?, SHA2(?, 256), TRUE, FALSE, NOW())"
		result, err := tx.Exec(qry, name, name+password)
		if err != nil {
			// No error.
			if isDuplicated(err) {
				duplicated = true
				return nil
			}
			return err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		userID = uint64(id)

		return nil
	}
	if err = r.query(f); err != nil {
		return 0, false, err
	}

	return userID, duplicated, nil
}

func (r *MySQL) UpdateUser(id uint64, password *string, admin *bool) error {
	f := func(tx *sql.Tx) error {
		set := []string{}
		args := []interface{}{}

		if password != nil {
			set = append(set, "`password` = SHA2(CONCAT(`name`, ?), 256)")
			args = append(args, *password)
		}
		if admin != nil {
			set = append(set, "`admin` = ?")
			args = append(args, *admin)
		}
		if len(set) == 0 {
			return nil
		}

		qry := fmt.Sprintf("UPDATE `user` SET %v WHERE `id` = %v", strings.Join(set, ","), id)
		_, err := tx.Exec(qry, args...)

		return err
	}

	return r.query(f)
}

func (r *MySQL) ActivateUser(id uint64) error {
	f := func(tx *sql.Tx) error {
		qry := "UPDATE `user` SET `enabled` = TRUE WHERE `id` = ?"
		_, err := tx.Exec(qry, id)
		return err
	}

	return r.query(f)
}

func (r *MySQL) DeactivateUser(id uint64) error {
	f := func(tx *sql.Tx) error {
		qry := "UPDATE `user` SET `enabled` = FALSE WHERE `id` = ?"
		_, err := tx.Exec(qry, id)
		return err
	}

	return r.query(f)
}

func (r *MySQL) Groups(offset uint32, limit uint8) (group []api.Group, err error) {
	f := func(tx *sql.Tx) error {
		qry := "SELECT `id`, `name`, `timestamp` "
		qry += "FROM `group` "
		qry += "ORDER BY `id` DESC "
		qry += "LIMIT ?, ?"

		rows, err := tx.Query(qry, offset, limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		group = []api.Group{}
		for rows.Next() {
			v := api.Group{}
			if err := rows.Scan(&v.ID, &v.Name, &v.Timestamp); err != nil {
				return err
			}
			group = append(group, v)
		}

		return rows.Err()
	}
	if err = r.query(f); err != nil {
		return nil, err
	}

	return group, nil
}

func (r *MySQL) AddGroup(name string) (groupID uint64, duplicated bool, err error) {
	f := func(tx *sql.Tx) error {
		qry := "INSERT INTO `group` (`name`, `timestamp`) VALUES (?, NOW())"
		result, err := tx.Exec(qry, name)
		if err != nil {
			// No error.
			if isDuplicated(err) {
				duplicated = true
				return nil
			}
			return err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		groupID = uint64(id)

		return nil
	}
	if err = r.query(f); err != nil {
		return 0, false, err
	}

	return groupID, duplicated, nil
}

func (r *MySQL) UpdateGroup(id uint64, name string) (duplicated bool, err error) {
	f := func(tx *sql.Tx) error {
		qry := "UPDATE `group` SET `name` = ? WHERE `id` = ?"
		if _, err := tx.Exec(qry, name, id); err != nil {
			// No error.
			if isDuplicated(err) {
				duplicated = true
				return nil
			}
			return err
		}

		return nil
	}
	if err = r.query(f); err != nil {
		return false, err

	}

	return duplicated, nil
}

func (r *MySQL) RemoveGroup(id uint64) error {
	f := func(tx *sql.Tx) error {
		_, err := tx.Exec("DELETE FROM `group` WHERE `id` = ?", id)
		if err != nil && isForeignkeyErr(err) {
			return errors.New("failed to remove a group: it has child hosts that are being used by group")
		}

		return err
	}

	return r.query(f)
}

func (r *MySQL) MAC(ip net.IP) (mac net.HardwareAddr, ok bool, err error) {
	if ip == nil {
		panic("IP address is nil")
	}

	f := func(tx *sql.Tx) error {
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
		row, err := tx.Query(qry, ip.String(), ip.String())
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

	f := func(tx *sql.Tx) error {
		// Initial value.
		status = network.LocationUnregistered

		var portID sql.NullInt64
		qry := "SELECT `port_id` FROM `host` WHERE `mac` = ? ORDER BY `port_id` DESC LOCK IN SHARE MODE"
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

		return nil
	}
	if err := r.query(f); err != nil {
		return "", 0, network.LocationUnregistered, err
	}

	return dpid, port, status, nil
}

func (r *MySQL) Switches(offset uint32, limit uint8) (sw []api.Switch, err error) {
	f := func(tx *sql.Tx) error {
		qry := "SELECT `id`, `dpid`, `n_ports`, `first_port`, `first_printed_port`, `description` "
		qry += "FROM `switch` "
		qry += "ORDER BY `id` DESC "
		qry += "LIMIT ?, ?"

		rows, err := tx.Query(qry, offset, limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		sw = []api.Switch{}
		for rows.Next() {
			v := api.Switch{}
			if err := rows.Scan(&v.ID, &v.DPID, &v.NumPorts, &v.FirstPort, &v.FirstPrintedPort, &v.Description); err != nil {
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

func (r *MySQL) AddSwitch(dpid uint64, nPorts, firstPort, firstPrintedPort uint16, desc string) (swID uint64, duplicated bool, err error) {
	f := func(tx *sql.Tx) error {
		swID, err = r.addSwitch(tx, dpid, nPorts, firstPort, firstPrintedPort, desc)
		if err != nil {
			// No error.
			if isDuplicated(err) {
				duplicated = true
				return nil
			}

			return err
		}
		if err := r.addPorts(tx, swID, firstPort, nPorts); err != nil {
			return err
		}

		return nil
	}
	if err = r.query(f); err != nil {
		return 0, false, err
	}

	return swID, duplicated, nil
}

func (r *MySQL) addSwitch(tx *sql.Tx, dpid uint64, nPorts, firstPort, firstPrintedPort uint16, desc string) (swID uint64, err error) {
	qry := "INSERT INTO switch (dpid, n_ports, first_port, first_printed_port, description) VALUES (?, ?, ?, ?, ?)"
	result, err := tx.Exec(qry, dpid, nPorts, firstPort, firstPrintedPort, desc)
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

func (r *MySQL) RemoveSwitch(id uint64) error {
	f := func(tx *sql.Tx) error {
		_, err := tx.Exec("DELETE FROM `switch` WHERE `id` = ?", id)
		if err != nil && isForeignkeyErr(err) {
			return errors.New("failed to remove a switch: it has child hosts connected to this switch")
		}

		return err
	}

	return r.query(f)
}

func (r *MySQL) Networks(offset uint32, limit uint8) (network []api.Network, err error) {
	f := func(tx *sql.Tx) error {
		qry := "SELECT `id`, INET_NTOA(`address`), `mask` "
		qry += "FROM `network` "
		qry += "ORDER BY `address` ASC, `mask` ASC "
		qry += "LIMIT ?, ?"

		rows, err := tx.Query(qry, offset, limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		network = []api.Network{}
		for rows.Next() {
			v := api.Network{}
			if err := rows.Scan(&v.ID, &v.Address, &v.Mask); err != nil {
				return err
			}
			network = append(network, v)
		}

		return rows.Err()
	}
	if err = r.query(f); err != nil {
		return nil, err
	}

	return network, nil
}

func (r *MySQL) AddNetwork(addr net.IP, mask net.IPMask) (netID uint64, duplicated bool, err error) {
	f := func(tx *sql.Tx) error {
		netID, err = r.addNetwork(tx, addr, mask)
		if err != nil {
			// No error.
			if isDuplicated(err) {
				duplicated = true
				return nil
			}
			return err
		}
		if err := r.addIPAddrs(tx, netID, addr, mask); err != nil {
			return err
		}

		return nil
	}
	if err = r.query(f); err != nil {
		return 0, false, err
	}

	return netID, duplicated, nil
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

func (r *MySQL) RemoveNetwork(id uint64) error {
	f := func(tx *sql.Tx) error {
		_, err := tx.Exec("DELETE FROM `network` WHERE `id` = ?", id)
		if err != nil && isForeignkeyErr(err) {
			return errors.New("failed to remove a network: it has child IP addresses that are being used by hosts")
		}

		return err
	}

	return r.query(f)
}

func (r *MySQL) IPAddrs(networkID uint64) (address []api.IP, err error) {
	f := func(tx *sql.Tx) error {
		qry := "SELECT A.`id`, INET_NTOA(A.`address`), A.`used`, C.`description`, IFNULL(CONCAT(E.`description`, '/', D.`number` - E.`first_port` + E.`first_printed_port`), '') "
		qry += "FROM `ip` A "
		qry += "JOIN `network` B ON A.`network_id` = B.`id` "
		qry += "LEFT JOIN `host` C ON C.`ip_id` = A.`id` "
		qry += "LEFT JOIN `port` D ON D.`id` = C.`port_id` "
		qry += "LEFT JOIN `switch` E ON E.`id` = D.`switch_id` "
		qry += "WHERE A.`network_id` = ?"

		rows, err := tx.Query(qry, networkID)
		if err != nil {
			return err
		}
		defer rows.Close()

		address = []api.IP{}
		for rows.Next() {
			v := api.IP{}
			var host, port sql.NullString
			if err := rows.Scan(&v.ID, &v.Address, &v.Used, &host, &port); err != nil {
				return err
			}
			v.Host = host.String
			v.Port = port.String
			address = append(address, v)
		}

		return rows.Err()
	}
	if err = r.query(f); err != nil {
		return nil, err
	}

	return address, nil
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

func (r *MySQL) Hosts() (hosts []api.Host, err error) {
	f := func(tx *sql.Tx) error {
		qry := `SELECT A.id, CONCAT(INET_NTOA(B.address), '/', E.mask), 
				IFNULL(CONCAT(D.description, '/', C.number - D.first_port + D.first_printed_port), ''), 
				HEX(A.mac), A.description, A.last_updated_timestamp 
			FROM host A 
			JOIN ip B ON A.ip_id = B.id 
			LEFT JOIN port C ON A.port_id = C.id 
			LEFT JOIN switch D ON C.switch_id = D.id 
			JOIN network E ON B.network_id = E.id 
			ORDER by A.id DESC`
		rows, err := tx.Query(qry)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			v := api.Host{}
			var timestamp time.Time

			if err := rows.Scan(&v.ID, &v.IP, &v.Port, &v.MAC, &v.Description, &timestamp); err != nil {
				return err
			}

			// Parse the MAC address.
			mac, err := decodeMAC(v.MAC)
			if err != nil {
				return err
			}
			v.MAC = mac.String()
			// Check its freshness.
			if time.Now().Sub(timestamp) > discovery.ProbeInterval*2 {
				v.Stale = true
			}

			hosts = append(hosts, v)
		}

		return rows.Err()
	}
	if err = r.query(f); err != nil {
		return nil, err
	}

	return hosts, nil
}

func (r *MySQL) Host(id uint64) (host api.Host, ok bool, err error) {
	f := func(tx *sql.Tx) error {
		qry := `SELECT A.id, CONCAT(INET_NTOA(B.address), '/', E.mask), 
				IFNULL(CONCAT(D.description, '/', C.number - D.first_port + D.first_printed_port), ''), 
				HEX(A.mac), A.description, A.last_updated_timestamp 
			FROM host A 
			JOIN ip B ON A.ip_id = B.id 
			LEFT JOIN port C ON A.port_id = C.id 
			LEFT JOIN switch D ON C.switch_id = D.id 
			JOIN network E ON B.network_id = E.id 
			WHERE A.id = ?`
		row, err := tx.Query(qry, id)
		if err != nil {
			return err
		}
		defer row.Close()

		if !row.Next() {
			return nil
		}

		var timestamp time.Time
		if err := row.Scan(&host.ID, &host.IP, &host.Port, &host.MAC, &host.Description, &timestamp); err != nil {
			return err
		}

		// Parse the MAC address.
		mac, err := decodeMAC(host.MAC)
		if err != nil {
			return err
		}
		host.MAC = mac.String()
		// Check its freshness.
		if time.Now().Sub(timestamp) > discovery.ProbeInterval*2 {
			host.Stale = true
		}

		ok = true

		return nil
	}
	if err = r.query(f); err != nil {
		return api.Host{}, false, err
	}

	return host, ok, nil
}

func (r *MySQL) AddHost(ipID uint64, mac net.HardwareAddr, desc string) (hostID uint64, err error) {
	f := func(tx *sql.Tx) error {
		ok, err := isAvailableIP(tx, ipID)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("already used IP address")
		}
		hostID, err = addNewHost(tx, ipID, mac, desc)
		if err != nil {
			return err
		}

		return nil
	}
	if err = r.query(f); err != nil {
		return 0, err
	}

	return hostID, nil
}

func addNewHost(tx *sql.Tx, ipID uint64, mac net.HardwareAddr, desc string) (uint64, error) {
	qry := "INSERT INTO host (ip_id, mac, description, last_updated_timestamp) VALUES (?, UNHEX(?), ?, NOW())"
	result, err := tx.Exec(qry, ipID, normalizeMAC(mac.String()), desc)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	if err := updateARPTableEntryByHost(tx, uint64(id), false); err != nil {
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
	// Empty row?
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
	f := func(tx *sql.Tx) error {
		if err := updateARPTableEntryByHost(tx, id, true); err != nil {
			return err
		}

		result, err := tx.Exec("DELETE FROM host WHERE id = ?", id)
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
	f := func(tx *sql.Tx) error {
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
	// Empty row?
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

func (r *MySQL) TogglePortVIP(swDPID uint64, portNum uint16) (result []virtualip.Address, err error) {
	f := func(tx *sql.Tx) error {
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
			result = append(result, virtualip.Address{IP: v.address, MAC: mac})
		}

		return nil
	}
	if err = r.query(f); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *MySQL) ToggleDeviceVIP(swDPID uint64) (result []virtualip.Address, err error) {
	f := func(tx *sql.Tx) error {
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
			result = append(result, virtualip.Address{IP: v.address, MAC: mac})
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
	// Empty row?
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

	if err := updateARPTableEntryByVIP(tx, v.id, false); err != nil {
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
	// Empty row?
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

func (r *MySQL) VIPs() (result []api.VIP, err error) {
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

		result = append(result, api.VIP{
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
	f := func(tx *sql.Tx) error {
		qry := `SELECT A.id, CONCAT(INET_NTOA(B.address), '/', C.mask), A.active_host_id, A.standby_host_id, A.description 
			FROM vip A 
			JOIN ip B ON A.ip_id = B.id 
			JOIN network C ON C.id = B.network_id 
			ORDER BY A.id DESC`
		rows, err := tx.Query(qry)
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

func (r *MySQL) AddVIP(ipID, activeID, standbyID uint64, desc string) (id uint64, cidr string, err error) {
	f := func(tx *sql.Tx) error {
		ok, err := isAvailableIP(tx, ipID)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("already used IP address")
		}
		id, err = addNewVIP(tx, ipID, activeID, standbyID, desc)
		if err != nil {
			return err
		}
		cidr, err = getIP(tx, ipID)
		if err != nil {
			return err
		}

		return nil
	}
	if err = r.query(f); err != nil {
		return 0, "", err
	}

	return id, cidr, nil
}

func addNewVIP(tx *sql.Tx, ipID, activeID, standbyID uint64, desc string) (uint64, error) {
	qry := "INSERT INTO vip (ip_id, active_host_id, standby_host_id, description) VALUES (?, ?, ?, ?)"
	result, err := tx.Exec(qry, ipID, activeID, standbyID, desc)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	if err := updateARPTableEntryByVIP(tx, uint64(id), false); err != nil {
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
	// Empty row?
	if !row.Next() {
		return "", fmt.Errorf("unknown IP (ID=%v)", id)
	}

	if err := row.Scan(&cidr); err != nil {
		return "", err
	}

	return cidr, nil
}

func (r *MySQL) RemoveVIP(id uint64) (ok bool, err error) {
	f := func(tx *sql.Tx) error {
		if err := updateARPTableEntryByVIP(tx, id, true); err != nil {
			return err
		}

		result, err := tx.Exec("DELETE FROM vip WHERE id = ?", id)
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
	f := func(tx *sql.Tx) error {
		// NOTE: Do not include VIP addresses!
		qry := "SELECT IFNULL(INET_NTOA(B.`address`), '0.0.0.0') "
		qry += "FROM `host` A "
		qry += "JOIN `ip` B "
		qry += "ON A.`ip_id` = B.`id` "
		qry += "WHERE A.`port_id` IS NULL OR A.`last_updated_timestamp` < NOW() - INTERVAL ? SECOND"

		rows, err := tx.Query(qry, uint64(expiration.Seconds()))
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
	f := func(tx *sql.Tx) error {
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

	// Empty row?
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
	f := func(tx *sql.Tx) error {
		portID, err := portID(tx, swDPID, portNum)
		if err != nil {
			return err
		}

		qry := "UPDATE `host` SET `port_id` = NULL WHERE `port_id` = ?"
		if _, err := tx.Exec(qry, portID); err != nil {
			return err
		}

		return nil
	}

	return r.query(f)
}

// ResetHostLocationsByDevice sets NULL to the host locations that belong to the
// device specified by swDPID.
func (r *MySQL) ResetHostLocationsByDevice(swDPID uint64) error {
	f := func(tx *sql.Tx) error {
		qry := "UPDATE `host` A "
		qry += "JOIN `port` B ON A.`port_id` = B.`id` "
		qry += "JOIN `switch` C ON B.`switch_id` = C.`id` "
		qry += "SET A.`port_id` = NULL "
		qry += "WHERE C.`dpid` = ?"

		_, err := tx.Exec(qry, swDPID)
		if err != nil {
			return err
		}

		return nil
	}

	return r.query(f)
}

// Elect selects a new master as uid if there is a no existing master that has
// been updated within expiration. elected will be true if this uid has been
// elected as the new master or was already elected.
func (r *MySQL) Elect(uid string, expiration time.Duration) (elected bool, err error) {
	f := func(tx *sql.Tx) error {
		var name string
		var timestamp time.Time
		qry := "SELECT `name`, `timestamp` "
		qry += "FROM `election` "
		qry += "WHERE `type` = 'MASTER' "
		qry += "FOR UPDATE" // Lock the selected row even if there is a no exsiting one.
		err = tx.QueryRow(qry).Scan(&name, &timestamp)
		// Real error?
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// No existing master?
		if err == sql.ErrNoRows {
			// I am the newly elected master!
			qry = "INSERT INTO `election` (`name`, `type`, `timestamp`) "
			qry += "VALUES (?, 'MASTER', NOW())"
			if _, err := tx.Exec(qry, uid); err != nil {
				return err
			}
			elected = true
		} else {
			// Already elected or another stale master?
			if name == uid || time.Now().Sub(timestamp) > expiration {
				qry = "UPDATE `election` SET `name` = ?, `timestamp` = NOW() WHERE `type` = 'MASTER'"
				if _, err := tx.Exec(qry, uid); err != nil {
					return err
				}
				elected = true
			}
		}

		return nil
	}

	if err := r.query(f); err != nil {
		return false, err
	}

	return elected, nil
}

// MACAddrs returns all the registered MAC addresses.
func (r *MySQL) MACAddrs() (result []net.HardwareAddr, err error) {
	f := func(tx *sql.Tx) error {
		rows, err := tx.Query("SELECT HEX(`mac`) FROM `host`")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var mac string
			if err := rows.Scan(&mac); err != nil {
				return err
			}

			// Parse the MAC address.
			v, err := decodeMAC(mac)
			if err != nil {
				return err
			}

			result = append(result, v)
		}

		return rows.Err()
	}

	if err = r.query(f); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *MySQL) RenewARPTable() error {
	f := func(tx *sql.Tx) error {
		hosts, err := getHostARPEntries(tx)
		if err != nil {
			return err
		}
		vips, err := getVIPARPEntries(tx)
		if err != nil {
			return err
		}

		for _, v := range append(hosts, vips...) {
			if err := updateARPTableEntry(tx, v.IP, v.MAC); err != nil {
				return err
			}
		}

		return nil
	}

	return r.query(f)
}

type arpEntry struct {
	IP  string
	MAC string
}

func getHostARPEntries(tx *sql.Tx) (result []arpEntry, err error) {
	qry := "SELECT INET_NTOA(B.`address`), HEX(A.`mac`) FROM `host` A JOIN `ip` B ON A.`ip_id` = B.`id`"
	rows, err := tx.Query(qry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ip, mac string
		if err := rows.Scan(&ip, &mac); err != nil {
			return nil, err
		}
		result = append(result, arpEntry{IP: ip, MAC: mac})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func getVIPARPEntries(tx *sql.Tx) (result []arpEntry, err error) {
	qry := "SELECT INET_NTOA(B.`address`), HEX(C.`mac`) FROM `vip` A JOIN `ip` B ON A.`ip_id` = B.`id` JOIN `host` C ON A.`active_host_id` = C.`id`"
	rows, err := tx.Query(qry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ip, mac string
		if err := rows.Scan(&ip, &mac); err != nil {
			return nil, err
		}
		result = append(result, arpEntry{IP: ip, MAC: mac})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *MySQL) GetARPTable() (result []announcer.ARPTableEntry, err error) {
	f := func(tx *sql.Tx) error {
		rows, err := tx.Query("SELECT INET_NTOA(`ip`), HEX(`mac`) FROM `arp`")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var ip, mac string
			if err := rows.Scan(&ip, &mac); err != nil {
				return err
			}
			addr := net.ParseIP(ip)
			if addr == nil {
				return fmt.Errorf("invalid IP address: %v", ip)
			}
			hwAddr, err := decodeMAC(mac)
			if err != nil {
				return err
			}

			result = append(result, announcer.ARPTableEntry{IP: addr, MAC: hwAddr})
		}

		return rows.Err()
	}

	if err = r.query(f); err != nil {
		return nil, err
	}

	return result, nil
}

const zeroMAC = "000000000000"

func updateARPTableEntryByHost(tx *sql.Tx, id uint64, invalidate bool) error {
	var ip, mac string
	qry := "SELECT INET_NTOA(B.`address`), HEX(A.`mac`) "
	qry += "FROM `host` A "
	qry += "JOIN `ip` B ON A.`ip_id` = B.`id` "
	qry += "WHERE A.`id` = ?"
	if err := tx.QueryRow(qry, id).Scan(&ip, &mac); err != nil {
		return err
	}
	if invalidate == true {
		mac = zeroMAC
	}

	return updateARPTableEntry(tx, ip, mac)
}

func updateARPTableEntryByVIP(tx *sql.Tx, id uint64, invalidate bool) error {
	var ip, mac string
	qry := "SELECT INET_NTOA(B.`address`), HEX(C.`mac`) "
	qry += "FROM `vip` A "
	qry += "JOIN `ip` B ON A.`ip_id` = B.`id` "
	qry += "JOIN `host` C ON A.`active_host_id` = C.`id` "
	qry += "WHERE A.`id` = ?"
	if err := tx.QueryRow(qry, id).Scan(&ip, &mac); err != nil {
		return err
	}
	if invalidate == true {
		mac = zeroMAC
	}

	return updateARPTableEntry(tx, ip, mac)
}

func updateARPTableEntry(tx *sql.Tx, ip, mac string) error {
	qry := "INSERT INTO `arp` (`ip`, `mac`) VALUES (INET_ATON(?), UNHEX(?)) ON DUPLICATE KEY UPDATE `mac` = UNHEX(?)"
	if _, err := tx.Exec(qry, ip, mac, mac); err != nil {
		return err
	}
	logger.Debugf("updated ARP table entry: IP=%v, MAC=%v", ip, mac)

	return nil
}
