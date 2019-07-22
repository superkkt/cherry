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
	"unicode/utf8"

	"github.com/superkkt/cherry/api"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/davecgh/go-spew/spew"
)

type ComponentTransaction interface {
	// Components returns a list of registered components. Pagination limit can be 0 that means no pagination.
	Components(categoryID uint64, pagination Pagination) ([]*Component, error)
	AddComponent(requesterID, categoryID uint64, name string) (component *Component, duplicated bool, err error)
	// UpdateComponent updates name of a component specified by id and then returns information of the component. It returns nil if the component does not exist.
	UpdateComponent(requesterID, componentID uint64, name string) (component *Component, duplicated bool, err error)
	// RemoveComponent removes a component specified by id and then returns information of the component before removing. It returns nil if the component does not exist.
	RemoveComponent(requesterID, componentID uint64) (*Component, error)
}

type Component struct {
	ID        uint64
	Category  Category
	Name      string
	Timestamp time.Time
}

func (r *Component) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID        uint64   `json:"id"`
		Category  Category `json:"category"`
		Name      string   `json:"name"`
		Timestamp int64    `json:"timestamp"`
	}{
		ID:        r.ID,
		Category:  r.Category,
		Name:      r.Name,
		Timestamp: r.Timestamp.Unix(),
	})
}

func (r *API) listComponent(w api.ResponseWriter, req *rest.Request) {
	p := new(listComponentParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode params: %v", err.Error())})
		return
	}
	logger.Debugf("listComponent request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var component []*Component
	f := func(tx Transaction) (err error) {
		component, err = tx.Components(p.CategoryID, p.Pagination)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to query the component list: %v", err.Error())})
		return
	}
	logger.Debugf("queried component list: %v", spew.Sdump(component))

	w.Write(api.Response{Status: api.StatusOkay, Data: component})
}

type listComponentParam struct {
	SessionID  string
	CategoryID uint64
	Pagination Pagination
}

func (r *listComponentParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID  string     `json:"session_id"`
		CategoryID uint64     `json:"category_id"`
		Pagination Pagination `json:"pagination"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = listComponentParam(v)

	return r.validate()
}

func (r *listComponentParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.CategoryID == 0 {
		return errors.New("invalid category id")
	}

	return nil
}

func (r *API) addComponent(w api.ResponseWriter, req *rest.Request) {
	p := new(addComponentParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode params: %v", err.Error())})
		return
	}
	logger.Debugf("addComponent request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	session, ok := r.session.Get(p.SessionID)
	if ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var component *Component
	var duplicated bool
	f := func(tx Transaction) (err error) {
		component, duplicated, err = tx.AddComponent(session.(*User).ID, p.CategoryID, p.Name)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to add a new component: %v", err.Error())})
		return
	}

	if duplicated {
		w.Write(api.Response{Status: api.StatusDuplicated, Message: fmt.Sprintf("duplicated component: category_id=%v, name=%v", p.CategoryID, p.Name)})
		return
	}
	logger.Debugf("added component info: %v", spew.Sdump(component))

	w.Write(api.Response{Status: api.StatusOkay, Data: component})
}

type addComponentParam struct {
	SessionID  string
	CategoryID uint64
	Name       string
}

func (r *addComponentParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID  string `json:"session_id"`
		CategoryID uint64 `json:"category_id"`
		Name       string `json:"name"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = addComponentParam(v)

	return r.validate()
}

func (r *addComponentParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.CategoryID == 0 {
		return errors.New("invalid category id")
	}
	if utf8.RuneCountInString(r.Name) < 2 || utf8.RuneCountInString(r.Name) > 255 {
		return fmt.Errorf("invalid name: %v", r.Name)
	}

	return nil
}

func (r *API) updateComponent(w api.ResponseWriter, req *rest.Request) {
	p := new(updateComponentParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode params: %v", err.Error())})
		return
	}
	logger.Debugf("updateComponent request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	session, ok := r.session.Get(p.SessionID)
	if ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var component *Component
	var duplicated bool
	f := func(tx Transaction) (err error) {
		component, duplicated, err = tx.UpdateComponent(session.(*User).ID, p.ID, p.Name)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to update component info: %v", err.Error())})
		return
	}

	if duplicated {
		w.Write(api.Response{Status: api.StatusDuplicated, Message: fmt.Sprintf("duplicated component: %v", p.Name)})
		return
	}
	if component == nil {
		w.Write(api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found component to update: %v", p.ID)})
		return
	}
	logger.Debugf("updated component info: %v", spew.Sdump(component))

	w.Write(api.Response{Status: api.StatusOkay, Data: component})
}

type updateComponentParam struct {
	SessionID string
	ID        uint64
	Name      string
}

func (r *updateComponentParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
		Name      string `json:"name"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = updateComponentParam(v)

	return r.validate()
}

func (r *updateComponentParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("invalid component id")
	}
	if utf8.RuneCountInString(r.Name) < 2 || utf8.RuneCountInString(r.Name) > 255 {
		return fmt.Errorf("invalid name: %v", r.Name)
	}

	return nil
}

func (r *API) removeComponent(w api.ResponseWriter, req *rest.Request) {
	p := new(removeComponentParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode params: %v", err.Error())})
		return
	}
	logger.Debugf("removeComponent request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	session, ok := r.session.Get(p.SessionID)
	if ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var component *Component
	f := func(tx Transaction) (err error) {
		component, err = tx.RemoveComponent(session.(*User).ID, p.ID)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to remove component info: %v", err.Error())})
		return
	}

	if component == nil {
		w.Write(api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found component to remove: %v", p.ID)})
		return
	}
	logger.Debugf("removed component info: %v", spew.Sdump(component))

	w.Write(api.Response{Status: api.StatusOkay})
}

type removeComponentParam struct {
	SessionID string
	ID        uint64
}

func (r *removeComponentParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = removeComponentParam(v)

	return r.validate()
}

func (r *removeComponentParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("invalid component id")
	}

	return nil
}
