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

type VIP struct {
	ID          uint64 `json:"id"`
	IP          string `json:"ip"` // FIXME: Use a native type.
	ActiveHost  Host   `json:"active_host"`
	StandbyHost Host   `json:"standby_host"`
	Description string `json:"description"`
}

func (r *API) listVIP(w rest.ResponseWriter, req *rest.Request) {
	p := new(listVIPParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&api.Response{Status: api.StatusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("listVIP request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	vip, err := r.DB.VIPs(p.Offset, p.Limit)
	if err != nil {
		logger.Errorf("failed to query the VIP list: %v", err)
		w.WriteJson(&api.Response{Status: api.StatusInternalServerError, Message: err.Error()})
		return
	}
	logger.Debugf("queried VIP list: %v", spew.Sdump(vip))

	w.WriteJson(&api.Response{Status: api.StatusOkay, Data: vip})
}

type listVIPParam struct {
	SessionID string
	Offset    uint32
	Limit     uint8
}

func (r *listVIPParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		Offset    uint32 `json:"offset"`
		Limit     uint8  `json:"limit"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = listVIPParam(v)

	return r.validate()
}

func (r *listVIPParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.Limit == 0 {
		return errors.New("invalid limit")
	}

	return nil
}

func (r *API) addVIP(w rest.ResponseWriter, req *rest.Request) {
	p := new(addVIPParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&api.Response{Status: api.StatusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("addVIP request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	id, duplicated, err := r.DB.AddVIP(p.IPID, p.ActiveHostID, p.StandbyHostID, p.Description)
	if err != nil {
		logger.Errorf("failed to add a new VIP: %v", err)
		w.WriteJson(&api.Response{Status: api.StatusInternalServerError, Message: err.Error()})
		return
	}
	if duplicated {
		logger.Infof("duplicated VIP: ip_id=%v", p.IPID)
		w.WriteJson(&api.Response{Status: api.StatusDuplicated, Message: fmt.Sprintf("duplicated VIP: ip_id=%v", p.IPID)})
		return
	}
	logger.Debugf("added VIP info: %v", spew.Sdump(p))

	vip, err := r.DB.VIP(id)
	if err != nil {
		logger.Errorf("failed to query the VIP: %v", err)
		w.WriteJson(&api.Response{Status: api.StatusInternalServerError, Message: err.Error()})
		return
	}
	if vip == nil {
		logger.Criticalf("not found added VIP: %v", id)
		w.WriteJson(&api.Response{Status: api.StatusInternalServerError, Message: err.Error()})
		return
	}

	if err := r.announce(vip.IP, vip.ActiveHost.MAC); err != nil {
		// Ignore this error.
		logger.Errorf("failed to send ARP announcement: %v", err)
	}

	w.WriteJson(&api.Response{Status: api.StatusOkay, Data: id})
}

type addVIPParam struct {
	SessionID     string
	IPID          uint64
	ActiveHostID  uint64
	StandbyHostID uint64
	Description   string
}

func (r *addVIPParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID     string `json:"session_id"`
		IPID          uint64 `json:"ip_id"`
		ActiveHostID  uint64 `json:"active_host_id"`
		StandbyHostID uint64 `json:"standby_host_id"`
		Description   string `json:"description"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = addVIPParam(v)

	return r.validate()
}

func (r *addVIPParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ActiveHostID == 0 {
		return errors.New("invalid active host id")
	}
	if r.StandbyHostID == 0 {
		return errors.New("invalid standby host id")
	}
	if r.ActiveHostID == r.StandbyHostID {
		return errors.New("same host for the active and standby")
	}
	if len(r.Description) > 255 {
		return errors.New("too long description")
	}

	return nil
}

func (r *API) removeVIP(w rest.ResponseWriter, req *rest.Request) {
	p := new(removeVIPParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&api.Response{Status: api.StatusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("removeVIP request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	vip, err := r.DB.VIP(p.ID)
	if err != nil {
		logger.Errorf("failed to query the VIP: %v", err)
		w.WriteJson(&api.Response{Status: api.StatusInternalServerError, Message: err.Error()})
		return
	}
	if vip == nil {
		logger.Infof("unknown VIP: %v", p.ID)
		w.WriteJson(&api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("unknown VIP: %v", p.ID)})
		return
	}

	if err := r.DB.RemoveVIP(p.ID); err != nil {
		logger.Errorf("failed to remove VIP info: %v", err)
		w.WriteJson(&api.Response{Status: api.StatusInternalServerError, Message: err.Error()})
		return
	}
	logger.Debugf("removed VIP info: %v", spew.Sdump(p))

	if err := r.announce(vip.IP, "00:00:00:00:00:00"); err != nil {
		// Ignore this error.
		logger.Errorf("failed to send ARP announcement: %v", err)
	}

	w.WriteJson(&api.Response{Status: api.StatusOkay})
}

type removeVIPParam struct {
	SessionID string
	ID        uint64
}

func (r *removeVIPParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = removeVIPParam(v)

	return r.validate()
}

func (r *removeVIPParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("invalid VIP id")
	}

	return nil
}

func (r *API) toggleVIP(w rest.ResponseWriter, req *rest.Request) {
	p := new(toggleVIPParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&api.Response{Status: api.StatusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("toggleVIP request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	if err := r.DB.ToggleVIP(p.ID); err != nil {
		logger.Errorf("failed to swap active host and standby host: %v", err)
		w.WriteJson(&api.Response{Status: api.StatusInternalServerError, Message: err.Error()})
		return
	}
	logger.Debugf("toggled VIP info: %v", spew.Sdump(p))

	vip, err := r.DB.VIP(p.ID)
	if err != nil {
		logger.Errorf("failed to query the VIP: %v", err)
		w.WriteJson(&api.Response{Status: api.StatusInternalServerError, Message: err.Error()})
		return
	}
	if vip == nil {
		logger.Criticalf("not found toggled VIP: %v", p.ID)
		w.WriteJson(&api.Response{Status: api.StatusInternalServerError, Message: err.Error()})
		return
	}

	if err := r.announce(vip.IP, vip.ActiveHost.MAC); err != nil {
		// Ignore this error.
		logger.Errorf("failed to send ARP announcement: %v", err)
	}

	w.WriteJson(&api.Response{Status: api.StatusOkay})
}

type toggleVIPParam struct {
	SessionID string
	ID        uint64
}

func (r *toggleVIPParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = toggleVIPParam(v)

	return r.validate()
}

func (r *toggleVIPParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("invalid VIP id")
	}

	return nil
}
