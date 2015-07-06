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
	"fmt"
)

func (r *MySQL) createTables() error {
	if err := r.createNetworkTable(); err != nil {
		return fmt.Errorf("creating network DB table: %v", err)
	}
	if err := r.createIPTable(); err != nil {
		return fmt.Errorf("creating ip DB table: %v", err)
	}
	if err := r.createGatewayTable(); err != nil {
		return fmt.Errorf("creating gateway DB table: %v", err)
	}
	if err := r.createHostTable(); err != nil {
		return fmt.Errorf("creating host DB table: %v", err)
	}
	if err := r.createRouterTable(); err != nil {
		return fmt.Errorf("creating router DB table: %v", err)
	}
	if err := r.createACLTable(); err != nil {
		return fmt.Errorf("creating acl DB table: %v", err)
	}
	if err := r.createVIPTable(); err != nil {
		return fmt.Errorf("creating vip DB table: %v", err)
	}

	return nil
}

func (r *MySQL) createNetworkTable() error {
	qry := "CREATE TABLE IF NOT EXISTS `network` ("
	qry += " `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,"
	qry += " `address` int unsigned NOT NULL,"
	qry += " `mask` int unsigned NOT NULL,"
	qry += " PRIMARY KEY (`id`),"
	qry += " UNIQUE KEY `address` (`address`, `mask`)"
	qry += ") ENGINE=InnoDB DEFAULT CHARSET=utf8;"

	_, err := r.db.Exec(qry)
	return err
}

func (r *MySQL) createIPTable() error {
	qry := "CREATE TABLE IF NOT EXISTS `ip` ("
	qry += " `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,"
	qry += " `network_id` bigint(20) unsigned NOT NULL,"
	qry += " `address` int unsigned NOT NULL,"
	qry += " PRIMARY KEY (`id`),"
	qry += " FOREIGN KEY (`network_id`) REFERENCES `network`(`id`) ON UPDATE CASCADE ON DELETE RESTRICT,"
	qry += " UNIQUE KEY `address` (`address`)"
	qry += ") ENGINE=InnoDB DEFAULT CHARSET=utf8;"

	_, err := r.db.Exec(qry)
	return err
}

func (r *MySQL) createGatewayTable() error {
	qry := "CREATE TABLE IF NOT EXISTS `gateway` ("
	qry += " `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,"
	qry += " `mac` char(17) NOT NULL,"
	qry += " PRIMARY KEY (`id`),"
	qry += " UNIQUE KEY `mac` (`mac`)"
	qry += ") ENGINE=InnoDB DEFAULT CHARSET=utf8;"

	_, err := r.db.Exec(qry)
	return err
}

func (r *MySQL) createHostTable() error {
	qry := "CREATE TABLE IF NOT EXISTS `host` ("
	qry += " `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,"
	qry += " `ip_id` bigint(20) unsigned DEFAULT NULL,"
	qry += " `mac` char(17) NOT NULL,"
	qry += " PRIMARY KEY (`id`),"
	qry += " FOREIGN KEY (`ip_id`) REFERENCES `ip`(`id`) ON UPDATE CASCADE ON DELETE RESTRICT,"
	qry += " UNIQUE KEY `ip-mac` (`ip_id`, `mac`)"
	qry += ") ENGINE=InnoDB DEFAULT CHARSET=utf8;"

	_, err := r.db.Exec(qry)
	return err
}

func (r *MySQL) createRouterTable() error {
	qry := "CREATE TABLE IF NOT EXISTS `router` ("
	qry += " `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,"
	qry += " `ip_id` bigint(20) unsigned NOT NULL,"
	qry += " PRIMARY KEY (`id`),"
	qry += " FOREIGN KEY (`ip_id`) REFERENCES `ip`(`id`) ON UPDATE CASCADE ON DELETE RESTRICT"
	qry += ") ENGINE=InnoDB DEFAULT CHARSET=utf8;"

	_, err := r.db.Exec(qry)
	return err
}

func (r *MySQL) createACLTable() error {
	qry := "CREATE TABLE IF NOT EXISTS `acl` ("
	qry += " `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,"
	qry += " `network` int unsigned NOT NULL,"
	qry += " `mask` int unsigned NOT NULL,"
	qry += " PRIMARY KEY (`id`),"
	qry += " UNIQUE KEY `acl` (`network`, `mask`)"
	qry += ") ENGINE=InnoDB DEFAULT CHARSET=utf8;"

	_, err := r.db.Exec(qry)
	return err
}

func (r *MySQL) createVIPTable() error {
	qry := "CREATE TABLE IF NOT EXISTS `vip` ("
	qry += " `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,"
	qry += " `ip_id` bigint(20) unsigned NOT NULL,"
	qry += " `host_id` bigint(20) unsigned NOT NULL,"
	qry += " PRIMARY KEY (`id`),"
	qry += " FOREIGN KEY (`ip_id`) REFERENCES `ip`(`id`) ON UPDATE CASCADE ON DELETE RESTRICT,"
	qry += " FOREIGN KEY (`host_id`) REFERENCES `host`(`id`) ON UPDATE CASCADE ON DELETE RESTRICT,"
	qry += " UNIQUE KEY `vip` (`ip_id`, `host_id`)"
	qry += ") ENGINE=InnoDB DEFAULT CHARSET=utf8;"

	_, err := r.db.Exec(qry)
	return err
}
