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
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/superkkt/cherry/api/ui"
	"github.com/superkkt/cherry/network"
	"github.com/superkkt/cherry/northbound/app/announcer"
	"github.com/superkkt/cherry/northbound/app/dhcp"
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

func (r *MySQL) MAC(ip net.IP) (mac net.HardwareAddr, ok bool, err error) {
	if ip == nil {
		panic("IP address is nil")
	}

	f := func(tx *sql.Tx) error {
		// Union query from both vip and host tables
		qry := `(SELECT B.mac, B.enabled
			 FROM vip 
			 A JOIN host B ON A.active_host_id = B.id 
			 JOIN ip C ON A.ip_id = C.id 
			 WHERE C.address = INET_ATON(?)
		 	) 
			UNION ALL 
			(SELECT A.mac, A.enabled
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
		var enabled bool
		if err := row.Scan(&v, &enabled); err != nil {
			return err
		}
		if v == nil || len(v) != 6 {
			return errors.New("invalid MAC address")
		}
		if enabled == false {
			return nil
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

// GetUndiscoveredHosts returns IP addresses whose physical location is still
// undiscovered or staled more than expiration. result can be nil on empty result.
func (r *MySQL) GetUndiscoveredHosts(expiration time.Duration) (result []net.IPNet, err error) {
	f := func(tx *sql.Tx) error {
		// NOTE: Do not include VIP addresses!
		qry := "SELECT IFNULL(INET_NTOA(B.`address`), '0.0.0.0'), IFNULL(C.`mask`, 0) "
		qry += "FROM `host` A "
		qry += "JOIN `ip` B ON A.`ip_id` = B.`id` "
		qry += "JOIN `network` C ON B.`network_id` = C.`id` "
		qry += "WHERE A.`port_id` IS NULL OR A.`last_updated_timestamp` < NOW() - INTERVAL ? SECOND"

		rows, err := tx.Query(qry, uint64(expiration.Seconds()))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var addr string
			var mask int
			if err := rows.Scan(&addr, &mask); err != nil {
				return err
			}

			ip := net.ParseIP(addr)
			if ip == nil {
				return fmt.Errorf("invalid IP address: %v", addr)
			}
			if ip.IsUnspecified() {
				continue
			}

			netmask := net.CIDRMask(mask, 32)
			if netmask == nil {
				return fmt.Errorf("invalid network mask: IP=%v, Mask=%v", addr, mask)
			}

			result = append(result, net.IPNet{IP: ip, Mask: netmask})
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
	qry := "SELECT INET_NTOA(B.`address`), HEX(A.`mac`), A.`enabled` FROM `host` A JOIN `ip` B ON A.`ip_id` = B.`id`"
	rows, err := tx.Query(qry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ip, mac string
		var enabled bool
		if err := rows.Scan(&ip, &mac, &enabled); err != nil {
			return nil, err
		}
		if enabled == false {
			continue
		}
		result = append(result, arpEntry{IP: ip, MAC: mac})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func getVIPARPEntries(tx *sql.Tx) (result []arpEntry, err error) {
	qry := "SELECT INET_NTOA(B.`address`), HEX(C.`mac`), C.`enabled` FROM `vip` A JOIN `ip` B ON A.`ip_id` = B.`id` JOIN `host` C ON A.`active_host_id` = C.`id`"
	rows, err := tx.Query(qry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ip, mac string
		var enabled bool
		if err := rows.Scan(&ip, &mac, &enabled); err != nil {
			return nil, err
		}
		if enabled == false {
			continue
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
		mac = encodeMAC(network.NullMAC)
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
		mac = encodeMAC(network.NullMAC)
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

func (r *MySQL) DHCP(mac net.HardwareAddr) (conf *dhcp.NetConfig, err error) {
	f := func(tx *sql.Tx) error {
		var ip, gateway string
		var mask int

		qry := "SELECT INET_NTOA(B.`address`), C.`mask`, INET_NTOA(C.`gateway`) "
		qry += "FROM `host` A "
		qry += "JOIN `ip` B ON A.`ip_id` = B.`id` "
		qry += "JOIN `network` C ON B.`network_id` = C.`id` "
		qry += "WHERE A.`mac` = ?"
		if err := tx.QueryRow(qry, []byte(mac)).Scan(&ip, &mask, &gateway); err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}

		v := &dhcp.NetConfig{
			IP:      net.ParseIP(ip),
			Mask:    net.CIDRMask(mask, net.IPv4len*8),
			Gateway: net.ParseIP(gateway),
		}
		if v.IP == nil || v.Mask == nil || v.Gateway == nil {
			return fmt.Errorf("invalid DHCP network configuration: MAC=%v, IP=%v, mask=%v, gateway=%v", mac, ip, mask, gateway)
		}
		if v.IP.Mask(v.Mask).Equal(v.Gateway.Mask(v.Mask)) == false {
			return fmt.Errorf("invalid DHCP network configuration: invalid gateway address: MAC=%v, IP=%v, mask=%v, gateway=%v", mac, ip, mask, gateway)
		}
		conf = v

		return nil
	}

	if err := r.query(f); err != nil {
		return nil, err
	}

	return conf, nil
}

// Exec executes all queries of f in a single transaction. f should return the error raised from the ui.Transaction
// without any change or wrapping it for deadlock protection.
func (r *MySQL) Exec(f func(ui.Transaction) error) error {
	deadlockRetry := 0

	for {
		tx, err := r.db.Begin()
		if err != nil {
			return err
		}

		err = f(&uiTx{handle: tx})
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

type uiTx struct {
	handle *sql.Tx
}

func (r *uiTx) Groups(pagination ui.Pagination) (group []*ui.Group, err error) {
	qry := "SELECT `id`, `name`, `timestamp` "
	qry += "FROM `group` "
	qry += "ORDER BY `id` DESC "
	if pagination.Limit > 0 {
		qry += fmt.Sprintf("LIMIT %v, %v", pagination.Offset, pagination.Limit)
	}

	rows, err := r.handle.Query(qry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	group = []*ui.Group{}
	for rows.Next() {
		v := new(ui.Group)
		if err := rows.Scan(&v.ID, &v.Name, &v.Timestamp); err != nil {
			return nil, err
		}
		group = append(group, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return group, nil
}

func (r *uiTx) AddGroup(requesterID uint64, name string) (group *ui.Group, duplicated bool, err error) {
	qry := "INSERT INTO `group` (`name`, `timestamp`) VALUES (?, NOW())"
	result, err := r.handle.Exec(qry, name)
	if err != nil {
		if isDuplicated(err) {
			return nil, true, nil
		}
		return nil, false, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, false, err
	}
	group, err = getGroup(r.handle, uint64(id))
	if err != nil {
		return nil, false, err
	}

	if err := r.log(requesterID, logTypeGroup, logMethodAdd, group); err != nil {
		return nil, false, err
	}

	return group, false, nil
}

func getGroup(tx *sql.Tx, id uint64) (*ui.Group, error) {
	qry := "SELECT `id`, `name`, `timestamp` "
	qry += "FROM `group` "
	qry += "WHERE `id` = ?"

	v := new(ui.Group)
	if err := tx.QueryRow(qry, id).Scan(&v.ID, &v.Name, &v.Timestamp); err != nil {
		return nil, err
	}

	return v, nil
}

func (r *uiTx) UpdateGroup(requesterID, groupID uint64, name string) (group *ui.Group, duplicated bool, err error) {
	qry := "UPDATE `group` SET `name` = ? WHERE `id` = ?"
	result, err := r.handle.Exec(qry, name, groupID)
	if err != nil {
		if isDuplicated(err) {
			return nil, true, nil
		}
		return nil, false, err
	}
	nRows, err := result.RowsAffected()
	if err != nil {
		return nil, false, err
	}
	// Not found group to update.
	if nRows == 0 {
		return nil, false, nil
	}

	group, err = getGroup(r.handle, groupID)
	if err != nil {
		return nil, false, err
	}

	if err := r.log(requesterID, logTypeGroup, logMethodUpdate, group); err != nil {
		return nil, false, err
	}

	return group, false, nil
}

func (r *uiTx) RemoveGroup(requesterID, groupID uint64) (group *ui.Group, err error) {
	group, err = getGroup(r.handle, groupID)
	if err != nil {
		// Not found group to remove.
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if _, err := r.handle.Exec("DELETE FROM `group` WHERE `id` = ?", groupID); err != nil {
		if isForeignkeyErr(err) {
			return nil, errors.New("failed to remove a group: it has child hosts that are being used by group")
		}
		return nil, err
	}

	if err := r.log(requesterID, logTypeGroup, logMethodRemove, group); err != nil {
		return nil, err
	}

	return group, nil
}

func (r *uiTx) Hosts(search *ui.Search, sort ui.Sort, pagination ui.Pagination) (host []*ui.Host, err error) {
	qry, args := buildHostsQuery(search, sort, pagination)
	rows, err := r.handle.Query(qry, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	host = []*ui.Host{}
	for rows.Next() {
		v := new(ui.Host)
		var timestamp time.Time
		if err := rows.Scan(&v.ID, &v.IP, &v.Port, &v.Group, &v.MAC, &v.Description, &v.Enabled, &timestamp, &v.Timestamp); err != nil {
			return nil, err
		}

		// Parse the MAC address.
		mac, err := decodeMAC(v.MAC)
		if err != nil {
			return nil, err
		}
		v.MAC = mac.String()
		// Check its freshness.
		if time.Now().Sub(timestamp) > discovery.ProbeInterval*2 {
			v.Stale = true
		}

		host = append(host, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, h := range host {
		spec, err := getSpec(r.handle, h.ID)
		if err != nil {
			return nil, err
		}
		h.Spec = spec
	}

	return host, nil
}

func buildHostsQuery(search *ui.Search, sort ui.Sort, pagination ui.Pagination) (qry string, args []interface{}) {
	qry = "SELECT `host`.`id`, "                                                                                                              // ID
	qry += "      CONCAT(INET_NTOA(`ip`.`address`), '/', `network`.`mask`), "                                                                 // IP
	qry += "      IFNULL(CONCAT(`switch`.`description`, '/', `port`.`number` - `switch`.`first_port` + `switch`.`first_printed_port`), ''), " // Port
	qry += "      IFNULL(`group`.`name`, ''), "                                                                                               // Group
	qry += "      HEX(`host`.`mac`), "                                                                                                        // MAC
	qry += "      `host`.`description`, "                                                                                                     // Description
	qry += "      `host`.`enabled`, "                                                                                                         // Enabled
	qry += "      `host`.`last_updated_timestamp`, "                                                                                          // Stale
	qry += "      `host`.`timestamp` "                                                                                                        // Timestamp
	qry += "FROM `host` "
	qry += "JOIN `ip` ON `host`.`ip_id` = `ip`.`id` "
	qry += "LEFT JOIN `port` ON `host`.`port_id` = `port`.`id` "
	qry += "LEFT JOIN `switch` ON `port`.`switch_id` = `switch`.`id` "
	qry += "JOIN `network` ON `ip`.`network_id` = `network`.`id` "
	qry += "LEFT JOIN `group` ON `host`.`group_id` = `group`.`id` "

	if search != nil {
		switch search.Key {
		// Query by description supports full text search.
		case ui.ColumnDescription:
			qry += fmt.Sprintf("WHERE MATCH (`host`.`description`) AGAINST (CONCAT(?, '*') IN BOOLEAN MODE) ")
			args = append(args, search.Value)
		case ui.ColumnPort:
			qry += fmt.Sprintf("WHERE `switch`.`description` LIKE CONCAT(?, '%%') ")
			args = append(args, search.Value)
		case ui.ColumnGroup:
			qry += fmt.Sprintf("WHERE `group`.`name` LIKE CONCAT(?, '%%') ")
			args = append(args, search.Value)
		case ui.ColumnIP:
			start, end := rangeIP(search.Value)
			qry += fmt.Sprintf("WHERE `ip`.`address` BETWEEN %v AND %v ", start, end)
		case ui.ColumnMAC:
			start, end := rangeMAC(search.Value)
			qry += fmt.Sprintf("WHERE `host`.`mac` BETWEEN UNHEX('%v') AND UNHEX('%v') ", start, end)
		default:
			panic(fmt.Sprintf("invalid search key: %v", search.Key))
		}
	}

	v := []string{}
	switch sort.Key {
	case ui.ColumnTime:
		// Do nothing. It's the default order.
	case ui.ColumnIP:
		v = append(v, "`ip`.`address`")
	case ui.ColumnMAC:
		v = append(v, "`host`.`mac`")
	case ui.ColumnGroup:
		v = append(v, "`group`.`name`")
	case ui.ColumnPort:
		v = append(v, "`port`.`switch_id`")
		v = append(v, "`port`.`number`")
	default:
		panic(fmt.Sprintf("invalid sort key: %v", sort.Key))
	}
	// If duplicated value is existed, sort by id.
	v = append(v, "`host`.`id`")

	switch sort.Order {
	case ui.OrderAscending:
		qry += fmt.Sprintf("ORDER BY %v ASC ", strings.Join(v, " ASC, "))
	case ui.OrderDescending:
		qry += fmt.Sprintf("ORDER BY %v DESC ", strings.Join(v, " DESC, "))
	default:
		panic(fmt.Sprintf("invalid sort order: %v", sort.Order))
	}

	if pagination.Limit > 0 {
		qry += fmt.Sprintf("LIMIT %v, %v", pagination.Offset, pagination.Limit)
	}

	return qry, args
}

// IP format is '1.*.*.*', '1.2.*.*', '1.2.3.*', '1.2.3.4'.
func rangeIP(ip string) (start, end uint32) {
	s := net.ParseIP(strings.Replace(ip, "*", "0", -1))
	if s == nil {
		panic("this ip should be parsed")
	}
	e := net.ParseIP(strings.Replace(ip, "*", "255", -1))
	if e == nil {
		panic("this ip should be parsed")
	}

	return convertIPToInt(s), convertIPToInt(e)
}

func convertIPToInt(ip net.IP) uint32 {
	v := ip.To4()
	if v == nil {
		panic("unexpected IP address format")
	}
	return binary.BigEndian.Uint32(v)
}

// MAC format is 'A1:*:*:*:*:*', 'A1:A2:*:*:*:*', 'A1:A2:A3:*:*:*', 'A1:A2:A3:A4:*:*', 'A1:A2:A3:A4:A5:*', 'A1:A2:A3:A4:A5:A6'.
func rangeMAC(mac string) (start, end string) {
	s, err := net.ParseMAC(strings.Replace(mac, "*", "00", -1))
	if err != nil {
		panic("this mac should be parsed")
	}
	e, err := net.ParseMAC(strings.Replace(mac, "*", "FF", -1))
	if err != nil {
		panic("this mac should be parsed")
	}

	return encodeMAC(s), encodeMAC(e)
}

func (r *uiTx) Host(id uint64) (host *ui.Host, err error) {
	h, err := getHost(r.handle, id)
	if err != nil {
		// Ignore the no rows error.
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return h.convert(), nil
}

func getHost(tx *sql.Tx, id uint64) (*host, error) {
	qry := "SELECT `host`.`id`, "                                                                                                              // ID
	qry += "       `host`.`ip_id`, "                                                                                                           // IP ID
	qry += "       CONCAT(INET_NTOA(`ip`.`address`), '/', `network`.`mask`), "                                                                 // IP Address
	qry += "       IFNULL(CONCAT(`switch`.`description`, '/', `port`.`number` - `switch`.`first_port` + `switch`.`first_printed_port`), ''), " // Port
	qry += "       `host`.`group_id`, "                                                                                                        // Group ID
	qry += "       IFNULL(`group`.`name`, ''), "                                                                                               // Group Name
	qry += "       HEX(`host`.`mac`), "                                                                                                        // MAC
	qry += "       `host`.`description`, "                                                                                                     // Description
	qry += "       `host`.`enabled`, "                                                                                                         // Enabled
	qry += "       `host`.`last_updated_timestamp`, "                                                                                          // Stale
	qry += "       `host`.`timestamp` "                                                                                                        // Timestamp
	qry += "FROM `host` "
	qry += "JOIN `ip` ON `host`.`ip_id` = `ip`.`id` "
	qry += "LEFT JOIN `port` ON `host`.`port_id` = `port`.`id` "
	qry += "LEFT JOIN `switch` ON `port`.`switch_id` = `switch`.`id` "
	qry += "JOIN `network` ON `ip`.`network_id` = `network`.`id` "
	qry += "LEFT JOIN `group` ON `host`.`group_id` = `group`.`id` "
	qry += "WHERE `host`.`id` = ?"

	v := new(host)
	var timestamp time.Time
	var mac string
	if err := tx.QueryRow(qry, id).Scan(&v.id, &v.ip.id, &v.ip.address, &v.port, &v.group.id, &v.group.name, &mac, &v.description, &v.enabled, &timestamp, &v.timestamp); err != nil {
		return nil, err
	}

	// Parse the MAC address.
	var err error
	v.mac, err = decodeMAC(mac)
	if err != nil {
		return nil, err
	}
	// Check its freshness.
	if time.Now().Sub(timestamp) > discovery.ProbeInterval*2 {
		v.stale = true
	}

	spec, err := getSpec(tx, id)
	if err != nil {
		return nil, err
	}
	v.spec = spec

	return v, nil
}

type host struct {
	id uint64
	ip struct {
		id      uint64
		address string
	}
	port  string
	group struct {
		id   *uint64
		name string
	}
	mac         net.HardwareAddr
	description string
	enabled     bool
	stale       bool
	spec        []*ui.Spec
	timestamp   time.Time
}

func (r *host) convert() *ui.Host {
	return &ui.Host{
		ID:          r.id,
		IP:          r.ip.address,
		Port:        r.port,
		Group:       r.group.name,
		MAC:         r.mac.String(),
		Description: r.description,
		Enabled:     r.enabled,
		Stale:       r.stale,
		Spec:        r.spec,
		Timestamp:   r.timestamp,
	}
}

func (r *uiTx) AddHost(requesterID, ipID uint64, groupID *uint64, mac net.HardwareAddr, desc string, spec []ui.SpecParam) (host *ui.Host, duplicated bool, err error) {
	h, duplicated, err := addNewHost(r.handle, ipID, groupID, mac, desc, spec)
	if err != nil {
		return nil, false, err
	}
	if duplicated {
		return nil, true, nil
	}

	if err := r.log(requesterID, logTypeHost, logMethodAdd, h.convert()); err != nil {
		return nil, false, err
	}

	return h.convert(), false, nil
}

func addNewHost(tx *sql.Tx, ipID uint64, groupID *uint64, mac net.HardwareAddr, desc string, spec []ui.SpecParam) (host *host, duplicated bool, err error) {
	ok, err := isAvailableIP(tx, ipID)
	if err != nil {
		return nil, false, err
	}
	if ok == false {
		return nil, true, nil
	}

	qry := "INSERT INTO `host` (`ip_id`, `group_id`, `mac`, `description`, `last_updated_timestamp`, `enabled`, `timestamp`) VALUES (?, ?, UNHEX(?), ?, NOW(), TRUE, NOW())"
	result, err := tx.Exec(qry, ipID, groupID, encodeMAC(mac), desc)
	if err != nil {
		return nil, false, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, false, err
	}

	if spec != nil {
		for _, v := range spec {
			if err := addSpec(tx, uint64(id), v.ComponentID, v.Count); err != nil {
				return nil, false, err
			}
		}
	}

	if err := updateARPTableEntryByHost(tx, uint64(id), false); err != nil {
		return nil, false, err
	}

	host, err = getHost(tx, uint64(id))
	if err != nil {
		return nil, false, err
	}

	return host, false, nil
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

func encodeMAC(mac net.HardwareAddr) string {
	// Remove spaces and colons
	return strings.Replace(strings.Replace(mac.String(), ":", "", -1), " ", "", -1)
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

func (r *uiTx) UpdateHost(requesterID, hostID uint64, ipID, groupID *uint64, mac net.HardwareAddr, desc *string, spec []ui.SpecParam) (host *ui.Host, duplicated bool, err error) {
	old, err := getHost(r.handle, hostID)
	if err != nil {
		// Not found host to update.
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	new, duplicated, err := updateHost(r.handle, old, ipID, groupID, mac, desc, spec)
	if err != nil {
		return nil, false, err
	}
	if duplicated {
		return nil, true, nil
	}

	err = r.log(requesterID, logTypeHost, logMethodUpdate, &struct {
		Old *ui.Host `json:"old"`
		New *ui.Host `json:"new"`
	}{
		Old: old.convert(),
		New: new.convert(),
	})
	if err != nil {
		return nil, false, err
	}

	return new.convert(), false, nil
}

func updateHost(tx *sql.Tx, old *host, ipID, groupID *uint64, mac net.HardwareAddr, desc *string, spec []ui.SpecParam) (new *host, duplicated bool, err error) {
	if err := removeHost(tx, old.id); err != nil {
		return nil, false, err
	}

	if ipID == nil {
		ipID = &old.ip.id
	}
	if groupID == nil {
		groupID = old.group.id
	}
	if mac == nil {
		mac = old.mac
	}
	if desc == nil {
		desc = &old.description
	}
	if spec == nil {
		for _, v := range old.spec {
			spec = append(spec, ui.SpecParam{
				ComponentID: v.Component.ID,
				Count:       v.Count,
			})
		}
	}

	return addNewHost(tx, *ipID, groupID, mac, *desc, spec)
}

func (r *uiTx) ActivateHost(requesterID, hostID uint64) (host *ui.Host, err error) {
	qry := "UPDATE `host` SET `enabled` = TRUE WHERE `id` = ?"
	result, err := r.handle.Exec(qry, hostID)
	if err != nil {
		return nil, err
	}
	nRows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	// Not found host to activate.
	if nRows == 0 {
		return nil, nil
	}

	if err := updateARPTableEntryByHost(r.handle, hostID, false); err != nil {
		return nil, err
	}

	h, err := getHost(r.handle, hostID)
	if err != nil {
		return nil, err
	}

	if err := r.log(requesterID, logTypeHost, logMethodUpdate, h.convert()); err != nil {
		return nil, err
	}

	return h.convert(), nil
}

func (r *uiTx) DeactivateHost(requesterID, hostID uint64) (host *ui.Host, err error) {
	qry := "UPDATE `host` SET `enabled` = FALSE WHERE `id` = ?"
	result, err := r.handle.Exec(qry, hostID)
	if err != nil {
		return nil, err
	}
	nRows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	// Not found host to deactivate.
	if nRows == 0 {
		return nil, nil
	}

	if err := updateARPTableEntryByHost(r.handle, hostID, true); err != nil {
		return nil, err
	}

	h, err := getHost(r.handle, hostID)
	if err != nil {
		return nil, err
	}

	if err := r.log(requesterID, logTypeHost, logMethodUpdate, h.convert()); err != nil {
		return nil, err
	}

	return h.convert(), nil
}

func (r *uiTx) CountVIPByHostID(id uint64) (count uint64, err error) {
	qry := "SELECT COUNT(*) FROM `vip` WHERE `active_host_id` = ? OR `standby_host_id` = ?"
	if err := r.handle.QueryRow(qry, id, id).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (r *uiTx) RemoveHost(requesterID, hostID uint64) (host *ui.Host, err error) {
	h, err := getHost(r.handle, hostID)
	if err != nil {
		// Not found host to remove.
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if err := removeHost(r.handle, hostID); err != nil {
		return nil, err
	}

	if err := r.log(requesterID, logTypeHost, logMethodRemove, h.convert()); err != nil {
		return nil, err
	}

	return h.convert(), nil
}

func removeHost(tx *sql.Tx, id uint64) error {
	if err := updateARPTableEntryByHost(tx, id, true); err != nil {
		return err
	}

	_, err := tx.Exec("DELETE FROM host WHERE id = ?", id)
	if err != nil && isForeignkeyErr(err) {
		return errors.New("failed to remove a host: it has child VIP addresses")
	}

	return err
}

func (r *uiTx) IPAddrs(networkID uint64) (address []*ui.IP, err error) {
	qry := "SELECT A.`id`, "                                                                                        // ID
	qry += "       INET_NTOA(A.`address`), "                                                                        // Address
	qry += "       A.`used`, "                                                                                      // Used
	qry += "       C.`description`, "                                                                               // Host Description
	qry += "       C.`enabled`, "                                                                                   // Host Enabled
	qry += "       C.`last_updated_timestamp`, "                                                                    // Host Stale
	qry += "       IFNULL(CONCAT(E.`description`, '/', D.`number` - E.`first_port` + E.`first_printed_port`), '') " // Port
	qry += "FROM `ip` A "
	qry += "JOIN `network` B ON A.`network_id` = B.`id` "
	qry += "LEFT JOIN `host` C ON C.`ip_id` = A.`id` "
	qry += "LEFT JOIN `port` D ON D.`id` = C.`port_id` "
	qry += "LEFT JOIN `switch` E ON E.`id` = D.`switch_id` "
	qry += "WHERE A.`network_id` = ? "
	qry += "ORDER BY A.`address` ASC"

	rows, err := r.handle.Query(qry, networkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	address = []*ui.IP{}
	for rows.Next() {
		v := new(ui.IP)
		var desc, port sql.NullString
		var enabled sql.NullBool
		var timestamp *time.Time
		if err := rows.Scan(&v.ID, &v.Address, &v.Used, &desc, &enabled, &timestamp, &port); err != nil {
			return nil, err
		}

		v.Host.Description = desc.String
		v.Host.Enabled = enabled.Bool
		v.Port = port.String
		// Check its freshness.
		if timestamp != nil && (time.Now().Sub(*timestamp) > discovery.ProbeInterval*2) {
			v.Host.Stale = true
		}

		address = append(address, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return address, nil
}

func (r *uiTx) Networks(address *string, pagination ui.Pagination) (network []*ui.Network, err error) {
	qry := "SELECT `id`, INET_NTOA(`address`), `mask`, INET_NTOA(`gateway`) "
	qry += "FROM `network` "
	if address != nil {
		start, end := rangeIP(*address)
		qry += fmt.Sprintf("WHERE `address` BETWEEN %v AND %v ", start, end)
	}
	qry += "ORDER BY `address` ASC, `mask` ASC "
	if pagination.Limit > 0 {
		qry += fmt.Sprintf("LIMIT %v, %v", pagination.Offset, pagination.Limit)
	}

	rows, err := r.handle.Query(qry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	network = []*ui.Network{}
	for rows.Next() {
		v := new(ui.Network)
		if err := rows.Scan(&v.ID, &v.Address, &v.Mask, &v.Gateway); err != nil {
			return nil, err
		}
		network = append(network, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return network, nil
}

func (r *uiTx) AddNetwork(requesterID uint64, addr net.IP, mask net.IPMask, gateway net.IP) (network *ui.Network, duplicated bool, err error) {
	id, err := addNetwork(r.handle, addr, mask, gateway)
	if err != nil {
		// No error.
		if isDuplicated(err) {
			return nil, true, nil
		}
		return nil, false, err
	}
	if err := addIPAddrs(r.handle, id, addr, mask); err != nil {
		return nil, false, err
	}

	network, err = getNetwork(r.handle, id)
	if err != nil {
		return nil, false, err
	}

	if err := r.log(requesterID, logTypeNetwork, logMethodAdd, network); err != nil {
		return nil, false, err
	}

	return network, false, nil
}

func addNetwork(tx *sql.Tx, addr net.IP, mask net.IPMask, gateway net.IP) (netID uint64, err error) {
	qry := "INSERT INTO network (address, mask, gateway) VALUES (INET_ATON(?), ?, INET_ATON(?))"
	ones, _ := mask.Size()
	result, err := tx.Exec(qry, addr.String(), ones, gateway.String())
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint64(id), nil
}

func addIPAddrs(tx *sql.Tx, netID uint64, addr net.IP, mask net.IPMask) error {
	stmt, err := tx.Prepare("INSERT INTO ip (network_id, address) VALUES (?, INET_ATON(?) + ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	reserved, err := network.ReservedIP(net.IPNet{IP: addr, Mask: mask})
	if err != nil {
		return err
	}

	ones, bits := mask.Size()
	n_addrs := int(math.Pow(2, float64(bits-ones))) - 2 // Minus two due to network and broadcast addresses
	for i := 0; i < n_addrs; i++ {
		// Skip the reserved IP address.
		if reserved.Equal(calculateIP(addr, uint32(i+1))) == true {
			continue
		}
		if _, err := stmt.Exec(netID, addr.String(), i+1); err != nil {
			return err
		}
	}

	return nil
}

func calculateIP(network net.IP, n uint32) net.IP {
	ip := network.To4()
	if ip == nil {
		panic(fmt.Sprintf("invalid IPv4 address: %v", network))
	}
	res := net.IPv4zero.To4()
	binary.BigEndian.PutUint32(res, binary.BigEndian.Uint32(ip)+n)

	return res
}

func getNetwork(tx *sql.Tx, id uint64) (*ui.Network, error) {
	qry := "SELECT `id`, INET_NTOA(`address`), `mask`, INET_NTOA(`gateway`) "
	qry += "FROM `network` "
	qry += "WHERE `id` = ?"

	v := new(ui.Network)
	if err := tx.QueryRow(qry, id).Scan(&v.ID, &v.Address, &v.Mask, &v.Gateway); err != nil {
		return nil, err
	}

	return v, nil
}

func (r *uiTx) RemoveNetwork(requesterID, netID uint64) (network *ui.Network, err error) {
	network, err = getNetwork(r.handle, netID)
	if err != nil {
		// Not found network to remove.
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if _, err := r.handle.Exec("DELETE FROM `network` WHERE `id` = ?", netID); err != nil {
		if isForeignkeyErr(err) {
			return nil, errors.New("failed to remove a network: it has child IP addresses that are being used by hosts")
		}
		return nil, err
	}

	if err := r.log(requesterID, logTypeNetwork, logMethodRemove, network); err != nil {
		return nil, err
	}

	return network, nil
}

func (r *uiTx) Switches(pagination ui.Pagination) (sw []*ui.Switch, err error) {
	qry := "SELECT `id`, `dpid`, `n_ports`, `first_port`, `first_printed_port`, `description` "
	qry += "FROM `switch` "
	qry += "ORDER BY `id` DESC "
	qry += "LIMIT ?, ?"

	rows, err := r.handle.Query(qry, pagination.Offset, pagination.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sw = []*ui.Switch{}
	for rows.Next() {
		v := new(ui.Switch)
		if err := rows.Scan(&v.ID, &v.DPID, &v.NumPorts, &v.FirstPort, &v.FirstPrintedPort, &v.Description); err != nil {
			return nil, err
		}
		sw = append(sw, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sw, nil
}

func (r *uiTx) AddSwitch(requesterID, dpid uint64, nPorts, firstPort, firstPrintedPort uint16, desc string) (sw *ui.Switch, duplicated bool, err error) {
	id, err := addSwitch(r.handle, dpid, nPorts, firstPort, firstPrintedPort, desc)
	if err != nil {
		// No error.
		if isDuplicated(err) {
			return nil, true, nil
		}

		return nil, false, err
	}
	if err := addPorts(r.handle, id, firstPort, nPorts); err != nil {
		return nil, false, err
	}

	sw, err = getSwitch(r.handle, id)
	if err != nil {
		return nil, false, err
	}

	if err := r.log(requesterID, logTypeSwitch, logMethodAdd, sw); err != nil {
		return nil, false, err
	}

	return sw, false, nil
}

func addSwitch(tx *sql.Tx, dpid uint64, nPorts, firstPort, firstPrintedPort uint16, desc string) (swID uint64, err error) {
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

func addPorts(tx *sql.Tx, swID uint64, firstPort, n_ports uint16) error {
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

func getSwitch(tx *sql.Tx, id uint64) (*ui.Switch, error) {
	qry := "SELECT `id`, `dpid`, `n_ports`, `first_port`, `first_printed_port`, `description` "
	qry += "FROM `switch` "
	qry += "WHERE `id` = ?"

	v := new(ui.Switch)
	if err := tx.QueryRow(qry, id).Scan(&v.ID, &v.DPID, &v.NumPorts, &v.FirstPort, &v.FirstPrintedPort, &v.Description); err != nil {
		return nil, err
	}

	return v, nil
}

func (r *uiTx) RemoveSwitch(requesterID, swID uint64) (sw *ui.Switch, err error) {
	sw, err = getSwitch(r.handle, swID)
	if err != nil {
		// Not found switch to remove.
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if _, err := r.handle.Exec("DELETE FROM `switch` WHERE `id` = ?", swID); err != nil {
		if isForeignkeyErr(err) {
			return nil, errors.New("failed to remove a switch: it has child hosts connected to this switch")
		}
		return nil, err
	}

	if err := r.log(requesterID, logTypeSwitch, logMethodRemove, sw); err != nil {
		return nil, err
	}

	return sw, nil
}

func (r *uiTx) User(name string) (user *ui.User, err error) {
	qry := "SELECT `id`, `name`, `key`, `enabled`, `admin`, `timestamp` "
	qry += "FROM `user` "
	qry += "WHERE `name` = ?"

	v := new(ui.User)
	if err := r.handle.QueryRow(qry, name).Scan(&v.ID, &v.Name, &v.Key, &v.Enabled, &v.Admin, &v.Timestamp); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return v, nil
}

func (r *uiTx) Users(pagination ui.Pagination) (user []*ui.User, err error) {
	qry := "SELECT `id`, `name`, `key`, `enabled`, `admin`, `timestamp` "
	qry += "FROM `user` "
	qry += "ORDER BY `id` DESC "
	qry += "LIMIT ?, ?"

	rows, err := r.handle.Query(qry, pagination.Offset, pagination.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	user = []*ui.User{}
	for rows.Next() {
		v := new(ui.User)
		if err := rows.Scan(&v.ID, &v.Name, &v.Key, &v.Enabled, &v.Admin, &v.Timestamp); err != nil {
			return nil, err
		}
		user = append(user, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return user, nil
}

func (r *uiTx) AddUser(requesterID uint64, name, key string) (user *ui.User, duplicated bool, err error) {
	qry := "INSERT INTO `user` (`name`, `key`, `enabled`, `admin`, `timestamp`) "
	qry += "VALUES (?, ?, TRUE, FALSE, NOW())"
	result, err := r.handle.Exec(qry, name, key)
	if err != nil {
		// No error.
		if isDuplicated(err) {
			return nil, true, nil
		}
		return nil, false, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, false, err
	}
	user, err = getUser(r.handle, uint64(id))
	if err != nil {
		return nil, false, err
	}

	if err := r.log(requesterID, logTypeUser, logMethodAdd, user); err != nil {
		return nil, false, err
	}

	return user, false, nil
}

func getUser(tx *sql.Tx, id uint64) (*ui.User, error) {
	qry := "SELECT `id`, `name`, `key`, `enabled`, `admin`, `timestamp` "
	qry += "FROM `user` "
	qry += "WHERE `id` = ?"

	v := new(ui.User)
	if err := tx.QueryRow(qry, id).Scan(&v.ID, &v.Name, &v.Key, &v.Enabled, &v.Admin, &v.Timestamp); err != nil {
		return nil, err
	}

	return v, nil
}

func (r *uiTx) UpdateUser(requesterID, userID uint64, enabled, admin *bool) (user *ui.User, err error) {
	set := []string{}
	args := []interface{}{}

	if enabled != nil {
		set = append(set, "`enabled` = ?")
		args = append(args, *enabled)
	}
	if admin != nil {
		set = append(set, "`admin` = ?")
		args = append(args, *admin)
	}
	if len(set) == 0 {
		return nil, nil
	}

	qry := fmt.Sprintf("UPDATE `user` SET %v WHERE `id` = %v", strings.Join(set, ","), userID)
	result, err := r.handle.Exec(qry, args...)
	if err != nil {
		return nil, err
	}
	nRows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	// Not found user to update.
	if nRows == 0 {
		return nil, nil
	}

	user, err = getUser(r.handle, userID)
	if err != nil {
		return nil, err
	}

	if err := r.log(requesterID, logTypeUser, logMethodUpdate, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (r *uiTx) ResetOTPKey(name, key string) (ok bool, err error) {
	result, err := r.handle.Exec("UPDATE `user` SET `key` = ? WHERE `name` = ?", key, name)
	if err != nil {
		return false, err
	}
	nRows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	// Not found user to reset OTP.
	if nRows == 0 {
		return false, nil
	}

	return true, nil
}

func (r *uiTx) VIPs(pagination ui.Pagination) (vip []*ui.VIP, err error) {
	reg, err := r.getVIPs(pagination)
	if err != nil {
		return nil, err
	}

	vip = []*ui.VIP{}
	for _, v := range reg {
		active, err := r.Host(v.active)
		if err != nil {
			return nil, err
		}
		if active == nil {
			return nil, fmt.Errorf("unknown active host: id=%v", v.active)
		}

		standby, err := r.Host(v.standby)
		if err != nil {
			return nil, err
		}
		if standby == nil {
			return nil, fmt.Errorf("unknown standby host: id=%v", v.standby)
		}

		vip = append(vip, &ui.VIP{
			ID:          v.id,
			IP:          v.address,
			ActiveHost:  *active,
			StandbyHost: *standby,
			Description: v.description,
		})
	}

	return vip, nil
}

type registeredVIP struct {
	id          uint64
	address     string
	active      uint64
	standby     uint64
	description string
}

func (r *uiTx) getVIPs(pagination ui.Pagination) (vip []registeredVIP, err error) {
	qry := "SELECT A.`id`, CONCAT(INET_NTOA(B.`address`), '/', C.`mask`), A.`active_host_id`, A.`standby_host_id`, A.`description` "
	qry += "FROM `vip` A "
	qry += "JOIN `ip` B ON A.`ip_id` = B.`id` "
	qry += "JOIN `network` C ON C.`id` = B.`network_id` "
	qry += "ORDER BY A.`id` DESC "
	qry += "LIMIT ?, ?"

	rows, err := r.handle.Query(qry, pagination.Offset, pagination.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vip = []registeredVIP{}
	for rows.Next() {
		v := registeredVIP{}
		if err := rows.Scan(&v.id, &v.address, &v.active, &v.standby, &v.description); err != nil {
			return nil, err
		}
		vip = append(vip, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return vip, nil
}

func (r *uiTx) AddVIP(requesterID, ipID, activeID, standbyID uint64, desc string) (vip *ui.VIP, duplicated bool, err error) {
	ok, err := isAvailableIP(r.handle, ipID)
	if err != nil {
		return nil, false, err
	}
	// No error.
	if ok == false {
		return nil, true, nil
	}

	id, err := addNewVIP(r.handle, ipID, activeID, standbyID, desc)
	if err != nil {
		return nil, false, err
	}
	vip, err = getVIP(r.handle, id)
	if err != nil {
		return nil, false, err
	}

	if err := r.log(requesterID, logTypeVIP, logMethodAdd, vip); err != nil {
		return nil, false, err
	}

	return vip, false, nil
}

func addNewVIP(tx *sql.Tx, ipID, activeID, standbyID uint64, desc string) (uint64, error) {
	enabled, err := isEnabledHost(tx, activeID)
	if err != nil {
		return 0, err
	}
	if enabled == false {
		return 0, errors.New("disabled host cannot be used for VIP")
	}

	enabled, err = isEnabledHost(tx, standbyID)
	if err != nil {
		return 0, err
	}
	if enabled == false {
		return 0, errors.New("disabled host cannot be used for VIP")
	}

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

func isEnabledHost(tx *sql.Tx, id uint64) (enabled bool, err error) {
	qry := "SELECT `enabled` FROM `host` WHERE `id` = ?"
	if err := tx.QueryRow(qry, id).Scan(&enabled); err != nil {
		return false, err
	}

	return enabled, nil
}

func getVIP(tx *sql.Tx, id uint64) (vip *ui.VIP, err error) {
	qry := "SELECT A.`id`, CONCAT(INET_NTOA(B.`address`), '/', C.`mask`), A.`active_host_id`, A.`standby_host_id`, A.`description` "
	qry += "FROM `vip` A "
	qry += "JOIN `ip` B ON A.`ip_id` = B.`id` "
	qry += "JOIN `network` C ON C.`id` = B.`network_id` "
	qry += "WHERE A.`id` = ?"

	v := new(registeredVIP)
	if err := tx.QueryRow(qry, id).Scan(&v.id, &v.address, &v.active, &v.standby, &v.description); err != nil {
		return nil, err
	}

	active, err := getHost(tx, v.active)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("unknown active host: id=%v", v.active)
		}
		return nil, err
	}
	standby, err := getHost(tx, v.standby)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("unknown standby host: id=%v", v.standby)
		}
		return nil, err
	}

	return &ui.VIP{
		ID:          v.id,
		IP:          v.address,
		ActiveHost:  *active.convert(),
		StandbyHost: *standby.convert(),
		Description: v.description,
	}, nil
}

func (r *uiTx) RemoveVIP(requesterID, vipID uint64) (vip *ui.VIP, err error) {
	vip, err = getVIP(r.handle, vipID)
	if err != nil {
		// Not found VIP to remove.
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if err := updateARPTableEntryByVIP(r.handle, vipID, true); err != nil {
		return nil, err
	}
	if _, err := r.handle.Exec("DELETE FROM `vip` WHERE `id` = ?", vipID); err != nil {
		return nil, err
	}

	if err := r.log(requesterID, logTypeVIP, logMethodRemove, vip); err != nil {
		return nil, err
	}

	return vip, nil
}

func (r *uiTx) ToggleVIP(requesterID, vipID uint64) (res *ui.VIP, err error) {
	v, err := getVIP(r.handle, vipID)
	if err != nil {
		// Not found VIP to toggle.
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	ip, _, err := net.ParseCIDR(v.IP)
	if err != nil {
		return nil, err
	}

	err = swapVIPHosts(r.handle, vip{
		id:      v.ID,
		address: ip,
		active:  v.ActiveHost.ID,
		standby: v.StandbyHost.ID,
	})
	if err != nil {
		return nil, err
	}

	res, err = getVIP(r.handle, vipID)
	if err != nil {
		return nil, err
	}

	if err := r.log(requesterID, logTypeVIP, logMethodUpdate, res); err != nil {
		return nil, err
	}

	return res, nil
}

func (r *uiTx) QueryLog(search *ui.Search, pagination ui.Pagination) (log []*ui.Log, err error) {
	qry, args := buildLogsQuery(search, pagination)
	rows, err := r.handle.Query(qry, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	log = []*ui.Log{}
	for rows.Next() {
		v := new(ui.Log)
		if err := rows.Scan(&v.ID, &v.User, &v.Type, &v.Method, &v.Data, &v.Timestamp); err != nil {
			return nil, err
		}
		log = append(log, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return log, nil
}

func buildLogsQuery(search *ui.Search, pagination ui.Pagination) (qry string, args []interface{}) {
	qry = "SELECT `log`.`id`, "       // ID
	qry += "      `user`.`name`, "    // User
	qry += "      `log`.`type`, "     // Type
	qry += "      `log`.`method`, "   // Method
	qry += "      `log`.`data`, "     // Data
	qry += "      `log`.`timestamp` " // Timestamp
	qry += "FROM `log` "
	qry += "JOIN `user` ON `log`.`user_id` = `user`.`id` "

	if search != nil {
		switch search.Key {
		case ui.ColumnUser:
			qry += fmt.Sprintf("WHERE `user`.`name` LIKE CONCAT(?, '%%') ")
		case ui.ColumnLogType:
			qry += fmt.Sprintf("WHERE `log`.`type` = ? ")
		case ui.ColumnLogMethod:
			qry += fmt.Sprintf("WHERE `log`.`method` = ? ")
		default:
			panic(fmt.Sprintf("invalid search key: %v", search.Key))
		}
		args = append(args, search.Value)
	}

	qry += "ORDER BY `log`.`id` DESC "
	qry += "LIMIT ?, ?"
	args = append(args, pagination.Offset)
	args = append(args, pagination.Limit)

	return qry, args
}

type logType int

const (
	logTypeInvalid logType = iota
	logTypeUser
	logTypeGroup
	logTypeSwitch
	logTypeNetwork
	logTypeHost
	logTypeVIP
	logTypeCategory
	logTypeComponent
)

func (r logType) validate() error {
	if r <= logTypeInvalid || r > logTypeComponent {
		return fmt.Errorf("invalid log type: %v", r)
	}

	return nil
}

type logMethod int

const (
	logMethodInvalid logMethod = iota
	logMethodAdd
	logMethodUpdate
	logMethodRemove
)

func (r logMethod) validate() error {
	if r <= logMethodInvalid || r > logMethodRemove {
		return fmt.Errorf("invalid log method: %v", r)
	}

	return nil
}

func (r *uiTx) log(userID uint64, t logType, m logMethod, data interface{}) error {
	if err := t.validate(); err != nil {
		return err
	}
	if err := m.validate(); err != nil {
		return err
	}

	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	qry := "INSERT INTO `log` (`user_id`, `type`, `method`, `data`, `timestamp`) VALUES (?, ?, ?, ?, NOW())"
	_, err = r.handle.Exec(qry, userID, t, m, b)
	return err
}

func (r *uiTx) Categories(pagination ui.Pagination) (category []*ui.Category, err error) {
	qry := "SELECT `id`, `name`, `timestamp` "
	qry += "FROM `category` "
	qry += "ORDER BY `id` DESC "
	if pagination.Limit > 0 {
		qry += fmt.Sprintf("LIMIT %v, %v", pagination.Offset, pagination.Limit)
	}

	rows, err := r.handle.Query(qry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	category = []*ui.Category{}
	for rows.Next() {
		v := new(ui.Category)
		if err := rows.Scan(&v.ID, &v.Name, &v.Timestamp); err != nil {
			return nil, err
		}
		category = append(category, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return category, nil
}

func (r *uiTx) AddCategory(requesterID uint64, name string) (category *ui.Category, duplicated bool, err error) {
	qry := "INSERT INTO `category` (`name`, `timestamp`) VALUES (?, NOW())"
	result, err := r.handle.Exec(qry, name)
	if err != nil {
		if isDuplicated(err) {
			return nil, true, nil
		}
		return nil, false, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, false, err
	}
	category, err = getCategory(r.handle, uint64(id))
	if err != nil {
		return nil, false, err
	}

	if err := r.log(requesterID, logTypeCategory, logMethodAdd, category); err != nil {
		return nil, false, err
	}

	return category, false, nil
}

func getCategory(tx *sql.Tx, id uint64) (category *ui.Category, err error) {
	qry := "SELECT `id`, `name`, `timestamp` "
	qry += "FROM `category` "
	qry += "WHERE `id` = ?"

	v := new(ui.Category)
	if err := tx.QueryRow(qry, id).Scan(&v.ID, &v.Name, &v.Timestamp); err != nil {
		return nil, err
	}

	return v, nil
}

func (r *uiTx) UpdateCategory(requesterID, categoryID uint64, name string) (category *ui.Category, duplicated bool, err error) {
	qry := "UPDATE `category` SET `name` = ? WHERE `id` = ?"
	result, err := r.handle.Exec(qry, name, categoryID)
	if err != nil {
		if isDuplicated(err) {
			return nil, true, nil
		}
		return nil, false, err
	}

	nRows, err := result.RowsAffected()
	if err != nil {
		return nil, false, err
	}
	// Not found category to update.
	if nRows == 0 {
		return nil, false, nil
	}

	category, err = getCategory(r.handle, categoryID)
	if err != nil {
		return nil, false, err
	}

	if err := r.log(requesterID, logTypeCategory, logMethodUpdate, category); err != nil {
		return nil, false, err
	}

	return category, false, nil
}

func (r *uiTx) RemoveCategory(requesterID, categoryID uint64) (category *ui.Category, err error) {
	category, err = getCategory(r.handle, categoryID)
	if err != nil {
		// Not found category to remove.
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if _, err = r.handle.Exec("DELETE FROM `category` WHERE `id` = ?", categoryID); err != nil {
		if isForeignkeyErr(err) {
			return nil, errors.New("failed to remove a category: it has child components that are being used by category")
		}
		return nil, err
	}

	if err := r.log(requesterID, logTypeCategory, logMethodRemove, category); err != nil {
		return nil, err
	}

	return category, nil
}

func (r *uiTx) Components(categoryID uint64, pagination ui.Pagination) (component []*ui.Component, err error) {
	qry := "SELECT A.`id`, A.`name`, A.`timestamp`, B.`id`, B.`name`, B.`timestamp` "
	qry += "FROM `component` A "
	qry += "JOIN `category` B ON A.`category_id` = B.`id` "
	qry += "WHERE B.`id`= ? "
	qry += "ORDER BY A.`id` DESC "
	if pagination.Limit > 0 {
		qry += fmt.Sprintf("LIMIT %v, %v", pagination.Offset, pagination.Limit)
	}

	rows, err := r.handle.Query(qry, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	component = []*ui.Component{}
	for rows.Next() {
		v := new(ui.Component)
		if err := rows.Scan(&v.ID, &v.Name, &v.Timestamp, &v.Category.ID, &v.Category.Name, &v.Category.Timestamp); err != nil {
			return nil, err
		}

		component = append(component, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return component, nil
}

func (r *uiTx) AddComponent(requesterID, categoryID uint64, name string) (component *ui.Component, duplicated bool, err error) {
	qry := "INSERT INTO `component` (`category_id`, `name`, `timestamp`) VALUES (?, ?, NOW())"
	result, err := r.handle.Exec(qry, categoryID, name)
	if err != nil {
		if isDuplicated(err) {
			return nil, true, nil
		}
		return nil, false, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, false, err
	}
	component, err = getComponent(r.handle, uint64(id))
	if err != nil {
		return nil, false, err
	}

	if err := r.log(requesterID, logTypeComponent, logMethodAdd, component); err != nil {
		return nil, false, err
	}

	return component, false, nil
}

func getComponent(tx *sql.Tx, id uint64) (component *ui.Component, err error) {
	qry := "SELECT A.`id`, A.`name`, A.`timestamp`, B.`id`, B.`name`, B.`timestamp` "
	qry += "FROM `component` A "
	qry += "JOIN `category` B ON A.`category_id` = B.`id` "
	qry += "WHERE A.`id` = ?"

	v := new(ui.Component)
	if err := tx.QueryRow(qry, id).Scan(&v.ID, &v.Name, &v.Timestamp, &v.Category.ID, &v.Category.Name, &v.Category.Timestamp); err != nil {
		return nil, err
	}

	return v, nil
}

func (r *uiTx) UpdateComponent(requesterID, componentID uint64, name string) (component *ui.Component, duplicated bool, err error) {
	qry := "UPDATE `component` SET `name` = ? WHERE `id` = ?"
	result, err := r.handle.Exec(qry, name, componentID)
	if err != nil {
		if isDuplicated(err) {
			return nil, true, nil
		}
		return nil, false, err
	}

	nRows, err := result.RowsAffected()
	if err != nil {
		return nil, false, err
	}
	// Not found component to update.
	if nRows == 0 {
		return nil, false, nil
	}

	component, err = getComponent(r.handle, componentID)
	if err != nil {
		return nil, false, err
	}

	if err := r.log(requesterID, logTypeComponent, logMethodUpdate, component); err != nil {
		return nil, false, err
	}

	return component, false, nil
}

func (r *uiTx) RemoveComponent(requesterID, componentID uint64) (component *ui.Component, err error) {
	component, err = getComponent(r.handle, componentID)
	if err != nil {
		// Not found component to remove.
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if _, err := r.handle.Exec("DELETE FROM `component` WHERE `id` = ?", componentID); err != nil {
		if isForeignkeyErr(err) {
			return nil, errors.New("failed to remove a component: it has child specs that are being used by component")
		}
		return nil, err
	}

	if err := r.log(requesterID, logTypeComponent, logMethodRemove, component); err != nil {
		return nil, err
	}

	return component, nil
}

func getSpec(tx *sql.Tx, hostID uint64) (spec []*ui.Spec, err error) {
	qry := "SELECT A.`id`, A.`count`, B.`id`, B.`name`, B.`timestamp`, C.`id`, C.`name`, C.`timestamp` "
	qry += "FROM `spec` A "
	qry += "JOIN `component` B ON A.`component_id` = B.`id` "
	qry += "JOIN `category` C ON B.`category_id` = C.`id` "
	qry += "WHERE A.`host_id` = ? "
	qry += "ORDER BY A.`id` DESC"

	rows, err := tx.Query(qry, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	spec = []*ui.Spec{}
	for rows.Next() {
		v := new(ui.Spec)
		c := ui.Component{}
		if err := rows.Scan(&v.ID, &v.Count, &c.ID, &c.Name, &c.Timestamp, &c.Category.ID, &c.Category.Name, &c.Category.Timestamp); err != nil {
			return nil, err
		}
		v.Component = c
		spec = append(spec, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return spec, nil
}

func addSpec(tx *sql.Tx, hostID, componentID uint64, count uint16) error {
	qry := "INSERT INTO `spec` (`host_id`, `component_id`, `count`) VALUES (?, ?, ?)"
	_, err := tx.Exec(qry, hostID, componentID, count)
	return err
}
