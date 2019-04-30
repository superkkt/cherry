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

package core

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/superkkt/cherry/api"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/davecgh/go-spew/spew"
	"github.com/superkkt/go-logging"
)

var (
	logger = logging.MustGetLogger("core")
)

type API struct {
	api.Server
}

func (r *API) Serve() error {
	return r.Server.Serve(
		rest.Post("/api/v1/status", api.ResponseHandler(r.status)),
		rest.Post("/api/v1/remove", api.ResponseHandler(r.remove)),
		rest.Post("/api/v1/announce", api.ResponseHandler(r.announce)),
	)
}

func (r *API) status(w api.ResponseWriter, req *rest.Request) {
	logger.Debugf("status request from %v", req.RemoteAddr)

	w.Write(api.Response{
		Status: api.StatusOkay,
		Data: struct {
			Master bool `json:"master"`
		}{
			Master: r.Observer.IsMaster(),
		},
	})
}

func (r *API) remove(w api.ResponseWriter, req *rest.Request) {
	p := new(removeParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("remove request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if p.MAC == nil {
		if err := r.Controller.RemoveFlows(); err != nil {
			w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to remove flows: %v", err.Error())})
			return
		}
	} else {
		if err := r.Controller.RemoveFlowsByMAC(p.MAC); err != nil {
			w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to remove flows by MAC: %v", err.Error())})
			return
		}
	}

	w.Write(api.Response{Status: api.StatusOkay})
}

type removeParam struct {
	MAC net.HardwareAddr
}

func (r *removeParam) UnmarshalJSON(data []byte) error {
	v := struct {
		MAC string `json:"mac"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	// If MAC is empty, remove all flows.
	if len(v.MAC) == 0 {
		return nil
	}

	addr, err := net.ParseMAC(v.MAC)
	if err != nil {
		return err
	}
	r.MAC = addr

	return nil
}

func (r *API) announce(w api.ResponseWriter, req *rest.Request) {
	p := new(announceParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("announce request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if err := r.Controller.Announce(p.IP, p.MAC); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to announce a new ARP entry: %v", err.Error())})
		return
	}

	w.Write(api.Response{Status: api.StatusOkay})
}

type announceParam struct {
	IP  net.IP
	MAC net.HardwareAddr
}

func (r *announceParam) UnmarshalJSON(data []byte) error {
	v := struct {
		IP  string `json:"ip"`
		MAC string `json:"mac"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	ip := net.ParseIP(v.IP)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %v", v.IP)
	}
	mac, err := net.ParseMAC(v.MAC)
	if err != nil {
		return err
	}
	r.IP = ip
	r.MAC = mac

	return nil
}
