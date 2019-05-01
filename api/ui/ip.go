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

package ui

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/superkkt/cherry/api"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/davecgh/go-spew/spew"
)

type IPTransaction interface {
	IPAddrs(networkID uint64) ([]IP, error)
}

type IP struct {
	ID      uint64 `json:"id"`
	Address string `json:"address"` // FIXME: Use a native type.
	Used    bool   `json:"used"`
	Port    string `json:"port"`
	Host    string `json:"host"`
}

func (r *API) listIP(w rest.ResponseWriter, req *rest.Request) {
	p := new(listIPParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&api.Response{Status: api.StatusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("listIP request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var ip []IP
	f := func(tx Transaction) (err error) {
		ip, err = tx.IPAddrs(p.NetworkID)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.WriteJson(&api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to query the network ip list: %v", err.Error())})
		return
	}
	logger.Debugf("queried network ip list: %v", spew.Sdump(ip))

	w.WriteJson(&api.Response{Status: api.StatusOkay, Data: ip})
}

type listIPParam struct {
	SessionID string
	NetworkID uint64
}

func (r *listIPParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		NetworkID uint64 `json:"network_id"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = listIPParam(v)

	return r.validate()
}

func (r *listIPParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.NetworkID == 0 {
		return errors.New("invalid network id")
	}

	return nil
}
