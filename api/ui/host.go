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
	"net"
	"time"

	"github.com/superkkt/cherry/api"
	"github.com/superkkt/cherry/network"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/davecgh/go-spew/spew"
)

var (
	errDuplicated = errors.New("duplicated error")
	errNotFound   = errors.New("not found error")
	errBlocked    = errors.New("blocked error")
)

type HostTransaction interface {
	Host(id uint64) (*Host, error)
	// Hosts returns a list of registered hosts. Search can be nil that means no search. Pagination limit can be 0 that means no pagination.
	Hosts(*Search, Sort, Pagination) ([]*Host, error)
	AddHost(requesterID, ipID uint64, groupID *uint64, mac net.HardwareAddr, desc string, spec []SpecParam) (host *Host, duplicated bool, err error)
	UpdateHost(requesterID, hostID, ipID uint64, groupID *uint64, mac net.HardwareAddr, desc string, spec []SpecParam) (host *Host, duplicated bool, err error)
	// ActivateHost enables a host specified by id and then returns information of the host. It returns nil if the host does not exist.
	ActivateHost(requesterID, hostID uint64) (*Host, error)
	// DeactivateHost disables a host specified by id and then returns information of the host. It returns nil if the host does not exist.
	DeactivateHost(requesterID, hostID uint64) (*Host, error)
	CountVIPByHostID(id uint64) (count uint64, err error)
	// RemoveHost removes a host specified by id and then returns information of the host before removing. It returns nil if the host does not exist.
	RemoveHost(requesterID, hostID uint64) (*Host, error)
}

type Host struct {
	ID          uint64
	IP          string // FIXME: Use a native type.
	Port        string
	Group       string
	MAC         string // FIXME: Use a native type.
	Description string
	Enabled     bool
	Stale       bool
	Spec        []*Spec
	Timestamp   time.Time
}

func (r *Host) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID          uint64  `json:"id"`
		IP          string  `json:"ip"`
		Port        string  `json:"port"`
		Group       string  `json:"group"`
		MAC         string  `json:"mac"`
		Description string  `json:"description"`
		Enabled     bool    `json:"enabled"`
		Stale       bool    `json:"stale"`
		Spec        []*Spec `json:"spec"`
		Timestamp   int64   `json:"timestamp"`
	}{
		ID:          r.ID,
		IP:          r.IP,
		Port:        r.Port,
		Group:       r.Group,
		MAC:         r.MAC,
		Description: r.Description,
		Enabled:     r.Enabled,
		Stale:       r.Stale,
		Spec:        r.Spec,
		Timestamp:   r.Timestamp.Unix(),
	})
}

type Spec struct {
	ID        uint64    `json:"id"`
	Component Component `json:"component"`
	Count     uint16    `json:"count"`
}

func (r *API) listHost(w api.ResponseWriter, req *rest.Request) {
	p := new(listHostParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("listHost request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var host []*Host
	f := func(tx Transaction) (err error) {
		host, err = tx.Hosts(p.Search, p.Sort, p.Pagination)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to query the hosts: %v", err.Error())})
		return
	}
	logger.Debugf("queried host list: %v", spew.Sdump(host))

	w.Write(api.Response{Status: api.StatusOkay, Data: host})
}

type listHostParam struct {
	SessionID  string
	Search     *Search
	Sort       Sort
	Pagination Pagination
}

func (r *listHostParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID  string     `json:"session_id"`
		Search     *Search    `json:"search"`
		Sort       Sort       `json:"sort"`
		Pagination Pagination `json:"pagination"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = listHostParam(v)

	return r.validate()
}

func (r *listHostParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	// If search is nil, fetch hosts without using search.
	if r.Search != nil {
		if r.Search.Key <= ColumnTime || r.Search.Key > ColumnDescription {
			return errors.New("invalid search key")
		}
		if err := r.Search.Validate(); err != nil {
			return err
		}
	}
	if err := r.Sort.Validate(); err != nil {
		return err
	}

	return nil
}

func (r *API) addHost(w api.ResponseWriter, req *rest.Request) {
	p := new(addHostParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("addHost request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	session, ok := r.session.Get(p.SessionID)
	if ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var host []*Host
	f := func(tx Transaction) (err error) {
		for _, v := range p.IPID {
			h, duplicated, err := tx.AddHost(session.(*User).ID, v, p.GroupID, p.MAC, p.Description, p.Spec)
			if err != nil {
				return err
			}
			if duplicated {
				return errDuplicated
			}

			host = append(host, h)
		}

		return nil
	}
	if err := r.DB.Exec(f); err != nil {
		if err == errDuplicated {
			w.Write(api.Response{Status: api.StatusDuplicated, Message: fmt.Sprintf("duplicated host: ip_id=%v", p.IPID)})
		} else {
			w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to add a new host: %v", err.Error())})
		}
		return
	}

	logger.Debugf("added host info: %v", spew.Sdump(host))

	for _, v := range host {
		if err := r.announce(v.IP, v.MAC); err != nil {
			// Ignore this error.
			logger.Errorf("failed to send ARP announcement: %v", err)
		}
	}

	w.Write(api.Response{Status: api.StatusOkay, Data: host})
}

type addHostParam struct {
	SessionID   string
	IPID        []uint64
	GroupID     *uint64
	MAC         net.HardwareAddr
	Description string
	Spec        []SpecParam
}

func (r *addHostParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID   string      `json:"session_id"`
		IPID        []uint64    `json:"ip_id"`
		GroupID     *uint64     `json:"group_id"`
		MAC         string      `json:"mac"`
		Description string      `json:"description"`
		Spec        []SpecParam `json:"spec"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	if len(v.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if len(v.IPID) == 0 {
		return errors.New("empty ip id")
	}
	if len(v.IPID) > 10 {
		return errors.New("too many ip ids")
	}
	for _, i := range v.IPID {
		if i == 0 {
			return errors.New("invalid ip id")
		}
	}
	if len(v.Description) > 255 {
		return errors.New("too long description")
	}
	mac, err := net.ParseMAC(v.MAC)
	if err != nil {
		return err
	}
	for _, v := range r.Spec {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	r.SessionID = v.SessionID
	r.IPID = v.IPID
	r.GroupID = v.GroupID
	r.MAC = mac
	r.Description = v.Description
	r.Spec = v.Spec

	return nil
}

type SpecParam struct {
	ComponentID uint64 `json:"component_id"`
	Count       uint16 `json:"count"`
}

func (r *SpecParam) Validate() error {
	if r.ComponentID == 0 {
		return errors.New("invalid component id")
	}
	if r.Count == 0 {
		return errors.New("invalid count")
	}

	return nil
}

func (r *API) updateHost(w api.ResponseWriter, req *rest.Request) {
	p := new(updateHostParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("updateHost request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	session, ok := r.session.Get(p.SessionID)
	if ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var old, new *Host
	f := func(tx Transaction) (err error) {
		count, err := tx.CountVIPByHostID(p.ID)
		if err != nil {
			return err
		}
		if count > 0 {
			return errors.New("VIP member host cannot be updated")
		}

		old, err = tx.Host(p.ID)
		if err != nil {
			return err
		}
		if old == nil {
			return errNotFound
		}
		if old.Enabled == false {
			return errBlocked
		}

		h, duplicated, err := tx.UpdateHost(session.(*User).ID, p.ID, p.IPID, p.GroupID, p.MAC, p.Description, p.Spec)
		if err != nil {
			return err
		}
		if duplicated {
			return errDuplicated
		}
		new = h

		return nil
	}
	if err := r.DB.Exec(f); err != nil {
		switch err {
		case errNotFound:
			w.Write(api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found host to update: %v", p.ID)})
		case errBlocked:
			w.Write(api.Response{Status: api.StatusBlockedHost, Message: fmt.Sprintf("unable to update blocked host: %v", p.ID)})
		case errDuplicated:
			w.Write(api.Response{Status: api.StatusDuplicated, Message: fmt.Sprintf("duplicated host: ip_id=%v", p.IPID)})
		default:
			w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to update a host: %v", err.Error())})
		}
		return
	}

	logger.Debugf("updated host info: %v", spew.Sdump(new))

	if err := r.announce(old.IP, network.NullMAC.String()); err != nil {
		// Ignore this error.
		logger.Errorf("failed to send ARP announcement: %v", err)
	}
	if err := r.announce(new.IP, new.MAC); err != nil {
		// Ignore this error.
		logger.Errorf("failed to send ARP announcement: %v", err)
	}

	w.Write(api.Response{Status: api.StatusOkay, Data: new})
}

type updateHostParam struct {
	SessionID   string
	ID          uint64
	IPID        uint64
	GroupID     *uint64
	MAC         net.HardwareAddr
	Description string
	Spec        []SpecParam
}

func (r *updateHostParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID   string      `json:"session_id"`
		ID          uint64      `json:"id"`
		IPID        uint64      `json:"ip_id"`
		GroupID     *uint64     `json:"group_id"`
		MAC         string      `json:"mac"`
		Description string      `json:"description"`
		Spec        []SpecParam `json:"spec"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	if len(v.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if v.ID == 0 {
		return errors.New("invalid host id")
	}
	if v.IPID == 0 {
		return errors.New("invalid ip id")
	}
	if len(v.Description) > 255 {
		return errors.New("too long description")
	}
	mac, err := net.ParseMAC(v.MAC)
	if err != nil {
		return err
	}
	for _, v := range r.Spec {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	r.SessionID = v.SessionID
	r.ID = v.ID
	r.IPID = v.IPID
	r.GroupID = v.GroupID
	r.MAC = mac
	r.Description = v.Description
	r.Spec = v.Spec

	return nil
}

func (r *API) activateHost(w api.ResponseWriter, req *rest.Request) {
	p := new(activateHostParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("activateHost request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	session, ok := r.session.Get(p.SessionID)
	if ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var host *Host
	f := func(tx Transaction) (err error) {
		host, err = tx.ActivateHost(session.(*User).ID, p.ID)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to activate a host: %v", err.Error())})
		return
	}

	if host == nil {
		w.Write(api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found host to activate: %v", p.ID)})
		return
	}
	logger.Debugf("activated host info: %v", spew.Sdump(host))

	if err := r.announce(host.IP, host.MAC); err != nil {
		// Ignore this error.
		logger.Errorf("failed to send ARP announcement: %v", err)
	}

	w.Write(api.Response{Status: api.StatusOkay})
}

type activateHostParam struct {
	SessionID string
	ID        uint64
}

func (r *activateHostParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = activateHostParam(v)

	return r.validate()
}

func (r *activateHostParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("invalid switch id")
	}

	return nil
}

func (r *API) deactivateHost(w api.ResponseWriter, req *rest.Request) {
	p := new(deactivateHostParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("deactivateHost request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	session, ok := r.session.Get(p.SessionID)
	if ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var host *Host
	f := func(tx Transaction) (err error) {
		count, err := tx.CountVIPByHostID(p.ID)
		if err != nil {
			return err
		}
		if count > 0 {
			return errors.New("VIP member host cannot be disabled")
		}

		host, err = tx.DeactivateHost(session.(*User).ID, p.ID)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to deactivate a host: %v", err.Error())})
		return
	}

	if host == nil {
		w.Write(api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found host to deactivate: %v", p.ID)})
		return
	}
	logger.Debugf("deactivated host info: %v", spew.Sdump(host))

	if err := r.announce(host.IP, network.NullMAC.String()); err != nil {
		// Ignore this error.
		logger.Errorf("failed to send ARP announcement: %v", err)
	}

	w.Write(api.Response{Status: api.StatusOkay})
}

type deactivateHostParam struct {
	SessionID string
	ID        uint64
}

func (r *deactivateHostParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = deactivateHostParam(v)

	return r.validate()
}

func (r *deactivateHostParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("invalid switch id")
	}

	return nil
}

func (r *API) removeHost(w api.ResponseWriter, req *rest.Request) {
	p := new(removeHostParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("removeHost request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	session, ok := r.session.Get(p.SessionID)
	if ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var host *Host
	f := func(tx Transaction) (err error) {
		host, err = tx.RemoveHost(session.(*User).ID, p.ID)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to remove a host: %v", err.Error())})
		return
	}

	if host == nil {
		w.Write(api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found host to remove: %v", p.ID)})
		return
	}
	logger.Debugf("removed host info: %v", spew.Sdump(host))

	if err := r.announce(host.IP, network.NullMAC.String()); err != nil {
		// Ignore this error.
		logger.Errorf("failed to send ARP announcement: %v", err)
	}

	w.Write(api.Response{Status: api.StatusOkay})
}

type removeHostParam struct {
	SessionID string
	ID        uint64
}

func (r *removeHostParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = removeHostParam(v)

	return r.validate()
}

func (r *removeHostParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("invalid switch id")
	}

	return nil
}
