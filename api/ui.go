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

package api

type UI struct {
	Config
}

func (r *UI) Serve() error {
	return r.serve(
	// TODO
	)
}

type Switch struct {
	ID               uint64 `json:"id"`
	DPID             uint64 `json:"dpid"`
	NumPorts         uint16 `json:"n_ports"`
	FirstPort        uint16 `json:"first_port"`
	FirstPrintedPort uint16 `json:"first_printed_port"`
	Description      string `json:"description"`
}

type SwitchPort struct {
	ID     uint64 `json:"id"`
	Number uint   `json:"number"`
}

type Network struct {
	ID      uint64 `json:"id"`
	Address string `json:"address"` // FIXME: Use a native type.
	Mask    uint8  `json:"mask"`    // FIXME: Use a native type.
}

type IP struct {
	ID      uint64 `json:"id"`
	Address string `json:"address"` // FIXME: Use a native type.
	Used    bool   `json:"used"`
	Port    string `json:"port"`
	Host    string `json:"host"`
}

type Host struct {
	ID          string `json:"id"`
	IP          string `json:"ip"` // FIXME: Use a native type.
	Port        string `json:"port"`
	MAC         string `json:"mac"` // FIXME: Use a native type.
	Description string `json:"description"`
	Stale       bool   `json:"stale"`
}

type VIP struct {
	ID          uint64 `json:"id"`
	IP          string `json:"ip"` // FIXME: Use a native type.
	ActiveHost  Host   `json:"active_host"`
	StandbyHost Host   `json:"standby_host"`
	Description string `json:"description"`
}
