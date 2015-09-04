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
	"fmt"
)

func createTables(db *sql.DB) error {
	if err := createSwitchTable(db); err != nil {
		return fmt.Errorf("creating network DB table: %v", err)
	}
	if err := createPortTable(db); err != nil {
		return fmt.Errorf("creating network DB table: %v", err)
	}
	if err := createNetworkTable(db); err != nil {
		return fmt.Errorf("creating network DB table: %v", err)
	}
	if err := createIPTable(db); err != nil {
		return fmt.Errorf("creating ip DB table: %v", err)
	}
	if err := createHostTable(db); err != nil {
		return fmt.Errorf("creating host DB table: %v", err)
	}
	if err := createACLTable(db); err != nil {
		return fmt.Errorf("creating acl DB table: %v", err)
	}
	if err := createVIPTable(db); err != nil {
		return fmt.Errorf("creating vip DB table: %v", err)
	}

	return nil
}

func createSwitchTable(db *sql.DB) error {
	qry := "CREATE TABLE IF NOT EXISTS `switch` ("
	qry += " `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,"
	qry += " `dpid` bigint(20) unsigned NOT NULL,"
	qry += " `n_ports` tinyint(3) unsigned NOT NULL,"
	qry += " `description` varchar(255) DEFAULT NULL,"
	qry += " PRIMARY KEY (`id`),"
	qry += " UNIQUE KEY `dpid` (`dpid`)"
	qry += ") ENGINE=InnoDB DEFAULT CHARSET=utf8;"

	_, err := db.Exec(qry)
	return err
}

func createPortTable(db *sql.DB) error {
	qry := "CREATE TABLE IF NOT EXISTS `port` ("
	qry += " `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,"
	qry += " `switch_id` bigint(20) unsigned NOT NULL,"
	qry += " `number` tinyint(3) unsigned NOT NULL,"
	qry += " `enabled` tinyint(1) NOT NULL DEFAULT '1',"
	qry += " PRIMARY KEY (`id`),"
	qry += " UNIQUE KEY `number` (`switch_id`,`number`),"
	qry += " FOREIGN KEY (`switch_id`) REFERENCES `switch` (`id`) ON UPDATE CASCADE ON DELETE RESTRICT"
	qry += ") ENGINE=InnoDB DEFAULT CHARSET=utf8;"

	_, err := db.Exec(qry)
	return err
}

func createNetworkTable(db *sql.DB) error {
	qry := "CREATE TABLE IF NOT EXISTS `network` ("
	qry += " `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,"
	qry += " `address` int unsigned NOT NULL,"
	qry += " `mask` int unsigned NOT NULL,"
	qry += " PRIMARY KEY (`id`),"
	qry += " UNIQUE KEY `address` (`address`, `mask`)"
	qry += ") ENGINE=InnoDB DEFAULT CHARSET=utf8;"

	_, err := db.Exec(qry)
	return err
}

func createIPTable(db *sql.DB) error {
	qry := "CREATE TABLE IF NOT EXISTS `ip` ("
	qry += " `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,"
	qry += " `network_id` bigint(20) unsigned NOT NULL,"
	qry += " `address` int unsigned NOT NULL,"
	qry += " PRIMARY KEY (`id`),"
	qry += " FOREIGN KEY (`network_id`) REFERENCES `network`(`id`) ON UPDATE CASCADE ON DELETE RESTRICT,"
	qry += " UNIQUE KEY `address` (`address`)"
	qry += ") ENGINE=InnoDB DEFAULT CHARSET=utf8;"

	_, err := db.Exec(qry)
	return err
}

func createHostTable(db *sql.DB) error {
	qry := "CREATE TABLE IF NOT EXISTS `host` ("
	qry += " `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,"
	qry += " `ip_id` bigint(20) unsigned NOT NULL,"
	qry += " `port_id` bigint(20) unsigned NOT NULL,"
	qry += " `mac` binary(6) NOT NULL,"
	qry += " `description` varchar(255) DEFAULT NULL,"
	qry += " PRIMARY KEY (`id`),"
	qry += " FOREIGN KEY (`ip_id`) REFERENCES `ip`(`id`) ON UPDATE CASCADE ON DELETE RESTRICT,"
	qry += " FOREIGN KEY (`port_id`) REFERENCES `port` (`id`) ON UPDATE CASCADE ON DELETE RESTRICT,"
	qry += " UNIQUE KEY `ip-port-mac` (`ip_id`, `port_id`, `mac`)"
	qry += ") ENGINE=InnoDB DEFAULT CHARSET=utf8;"

	_, err := db.Exec(qry)
	return err
}

func createACLTable(db *sql.DB) error {
	qry := "CREATE TABLE IF NOT EXISTS `acl` ("
	qry += " `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,"
	qry += " `network` int unsigned NOT NULL,"
	qry += " `mask` int unsigned NOT NULL,"
	qry += " PRIMARY KEY (`id`),"
	qry += " UNIQUE KEY `acl` (`network`, `mask`)"
	qry += ") ENGINE=InnoDB DEFAULT CHARSET=utf8;"

	_, err := db.Exec(qry)
	return err
}

func createVIPTable(db *sql.DB) error {
	qry := "CREATE TABLE IF NOT EXISTS `vip` ("
	qry += " `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,"
	qry += " `ip_id` bigint(20) unsigned NOT NULL,"
	qry += " `host_id` bigint(20) unsigned NOT NULL,"
	qry += " PRIMARY KEY (`id`),"
	qry += " FOREIGN KEY (`ip_id`) REFERENCES `ip`(`id`) ON UPDATE CASCADE ON DELETE RESTRICT,"
	qry += " FOREIGN KEY (`host_id`) REFERENCES `host`(`id`) ON UPDATE CASCADE ON DELETE RESTRICT,"
	qry += " UNIQUE KEY `vip` (`ip_id`, `host_id`)"
	qry += ") ENGINE=InnoDB DEFAULT CHARSET=utf8;"

	_, err := db.Exec(qry)
	return err
}
