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
	"time"

	"github.com/superkkt/cherry/api"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/davecgh/go-spew/spew"
)

type GroupTransaction interface {
	// Groups returns a list of registered groups. Pagination limit can be 0 that means no pagination.
	Groups(Pagination) ([]*Group, error)
	AddGroup(name string) (group *Group, duplicated bool, err error)
	// UpdateGroup updates name of a group specified by id and then returns information of the group. It returns nil if the group does not exist.
	UpdateGroup(id uint64, name string) (group *Group, duplicated bool, err error)
	// RemoveGroup removes a group specified by id and then returns information of the group before removing. It returns nil if the group does not exist.
	RemoveGroup(id uint64) (*Group, error)
}

type Group struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
}

func (r *Group) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID        uint64 `json:"id"`
		Name      string `json:"name"`
		Timestamp int64  `json:"timestamp"`
	}{
		ID:        r.ID,
		Name:      r.Name,
		Timestamp: r.Timestamp.Unix(),
	})
}

func (r *API) listGroup(w api.ResponseWriter, req *rest.Request) {
	p := new(listGroupParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("listGroup request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var group []*Group
	f := func(tx Transaction) (err error) {
		group, err = tx.Groups(p.Pagination)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to query the groups: %v", err.Error())})
		return
	}
	logger.Debugf("queried group list: %v", spew.Sdump(group))

	w.Write(api.Response{Status: api.StatusOkay, Data: group})
}

type listGroupParam struct {
	SessionID  string
	Pagination Pagination
}

func (r *listGroupParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID  string     `json:"session_id"`
		Pagination Pagination `json:"pagination"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = listGroupParam(v)

	return r.validate()
}

func (r *listGroupParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}

	return nil
}

func (r *API) addGroup(w api.ResponseWriter, req *rest.Request) {
	p := new(addGroupParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("addGroup request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var group *Group
	var duplicated bool
	f := func(tx Transaction) (err error) {
		group, duplicated, err = tx.AddGroup(p.Name)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to add a new group: %v", err.Error())})
		return
	}

	if duplicated {
		w.Write(api.Response{Status: api.StatusDuplicated, Message: fmt.Sprintf("duplicated group: %v", p.Name)})
		return
	}
	logger.Debugf("added group info: %v", spew.Sdump(group))

	w.Write(api.Response{Status: api.StatusOkay, Data: group})
}

type addGroupParam struct {
	SessionID string
	Name      string
}

func (r *addGroupParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		Name      string `json:"name"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = addGroupParam(v)

	return r.validate()
}

func (r *addGroupParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if len(r.Name) < 2 || len(r.Name) > 25 {
		return fmt.Errorf("invalid name: %v", r.Name)
	}

	return nil
}

func (r *API) updateGroup(w api.ResponseWriter, req *rest.Request) {
	p := new(updateGroupParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("updateGroup request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var group *Group
	var duplicated bool
	f := func(tx Transaction) (err error) {
		group, duplicated, err = tx.UpdateGroup(p.ID, p.Name)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to update a group: %v", err.Error())})
		return
	}

	if group == nil {
		w.Write(api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found group to update: %v", p.ID)})
		return
	}
	if duplicated {
		w.Write(api.Response{Status: api.StatusDuplicated, Message: fmt.Sprintf("duplicated group: %v", p.Name)})
		return
	}
	logger.Debugf("updated the group: %v", spew.Sdump(group))

	w.Write(api.Response{Status: api.StatusOkay, Data: group})
}

type updateGroupParam struct {
	SessionID string
	ID        uint64
	Name      string
}

func (r *updateGroupParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
		Name      string `json:"name"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = updateGroupParam(v)

	return r.validate()
}

func (r *updateGroupParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("invalid group id")
	}
	if len(r.Name) < 2 || len(r.Name) > 25 {
		return fmt.Errorf("invalid name: %v", r.Name)
	}

	return nil
}

func (r *API) removeGroup(w api.ResponseWriter, req *rest.Request) {
	p := new(removeGroupParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("removeGroup request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var group *Group
	f := func(tx Transaction) (err error) {
		group, err = tx.RemoveGroup(p.ID)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to remove a group: %v", err.Error())})
		return
	}

	if group == nil {
		w.Write(api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found group to remove: %v", p.ID)})
		return
	}
	logger.Debugf("removed the group: %v", spew.Sdump(group))

	w.Write(api.Response{Status: api.StatusOkay})
}

type removeGroupParam struct {
	SessionID string
	ID        uint64
}

func (r *removeGroupParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = removeGroupParam(v)

	return r.validate()
}

func (r *removeGroupParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("invalid group id")
	}

	return nil
}
