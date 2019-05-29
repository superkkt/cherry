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

type CategoryTransaction interface {
	// Categories returns a list of registered categories. Pagination limit can be 0 that means no pagination.
	Categories(Pagination) ([]*Category, error)
	AddCategory(requesterID uint64, name string) (category *Category, duplicated bool, err error)
	// UpdateCategory updates name of a category specified by id and then returns information of the category. It returns nil if the category does not exist.
	UpdateCategory(requesterID, categoryID uint64, name string) (category *Category, duplicated bool, err error)
	// RemoveCategory removes a category specified by id and then returns information of the category before removing. It returns nil if the category does not exist.
	RemoveCategory(requesterID, categoryID uint64) (*Category, error)
}

type Category struct {
	ID        uint64
	Name      string
	Timestamp time.Time
}

func (r *Category) MarshalJSON() ([]byte, error) {
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

func (r *API) listCategory(w api.ResponseWriter, req *rest.Request) {
	p := new(listCategoryParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode params: %v", err.Error())})
		return
	}
	logger.Debugf("listCategory request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var category []*Category
	f := func(tx Transaction) (err error) {
		category, err = tx.Categories(p.Pagination)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to query the category list: %v", err.Error())})
		return
	}
	logger.Debugf("queried category list: %v", spew.Sdump(category))

	w.Write(api.Response{Status: api.StatusOkay, Data: category})
}

type listCategoryParam struct {
	SessionID  string
	Pagination Pagination
}

func (r *listCategoryParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID  string     `json:"session_id"`
		Pagination Pagination `json:"pagination"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = listCategoryParam(v)

	return r.validate()
}

func (r *listCategoryParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}

	return nil
}

func (r *API) addCategory(w api.ResponseWriter, req *rest.Request) {
	p := new(addCategoryParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode params: %v", err.Error())})
		return
	}
	logger.Debugf("addCategory request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	session, ok := r.session.Get(p.SessionID)
	if ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var category *Category
	var duplicated bool
	f := func(tx Transaction) (err error) {
		category, duplicated, err = tx.AddCategory(session.(*User).ID, p.Name)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to add a new category: %v", err.Error())})
		return
	}

	if duplicated {
		w.Write(api.Response{Status: api.StatusDuplicated, Message: fmt.Sprintf("duplicated category: %v", p.Name)})
		return
	}
	logger.Debugf("added category info: %v", spew.Sdump(category))

	w.Write(api.Response{Status: api.StatusOkay, Data: category})
}

type addCategoryParam struct {
	SessionID string
	Name      string
}

func (r *addCategoryParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		Name      string `json:"name"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = addCategoryParam(v)

	return r.validate()
}

func (r *addCategoryParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if len(r.Name) < 2 || len(r.Name) > 255 {
		return fmt.Errorf("invalid name: %v", r.Name)
	}

	return nil
}

func (r *API) updateCategory(w api.ResponseWriter, req *rest.Request) {
	p := new(updateCategoryParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode params: %v", err.Error())})
		return
	}
	logger.Debugf("updateCategory request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	session, ok := r.session.Get(p.SessionID)
	if ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var category *Category
	var duplicated bool
	f := func(tx Transaction) (err error) {
		category, duplicated, err = tx.UpdateCategory(session.(*User).ID, p.ID, p.Name)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to update category info: %v", err.Error())})
		return
	}

	if duplicated {
		w.Write(api.Response{Status: api.StatusDuplicated, Message: fmt.Sprintf("duplicated category: %v", p.Name)})
		return
	}
	if category == nil {
		w.Write(api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found category to update: %v", p.ID)})
		return
	}
	logger.Debugf("updated category info: %v", spew.Sdump(category))

	w.Write(api.Response{Status: api.StatusOkay, Data: category})
}

type updateCategoryParam struct {
	SessionID string
	ID        uint64
	Name      string
}

func (r *updateCategoryParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
		Name      string `json:"name"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = updateCategoryParam(v)

	return r.validate()
}

func (r *updateCategoryParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("invalid category id")
	}
	if len(r.Name) < 2 || len(r.Name) > 255 {
		return fmt.Errorf("invalid name: %v", r.Name)
	}

	return nil
}

func (r *API) removeCategory(w api.ResponseWriter, req *rest.Request) {
	p := new(removeCategoryParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode params: %v", err.Error())})
		return
	}
	logger.Debugf("removeCategory request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	session, ok := r.session.Get(p.SessionID)
	if ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var category *Category
	f := func(tx Transaction) (err error) {
		category, err = tx.RemoveCategory(session.(*User).ID, p.ID)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to remove category info: %v", err.Error())})
		return
	}

	if category == nil {
		w.Write(api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found category to remove: %v", p.ID)})
		return
	}
	logger.Debugf("removed category info: %v", spew.Sdump(category))

	w.Write(api.Response{Status: api.StatusOkay})
}

type removeCategoryParam struct {
	SessionID string
	ID        uint64
}

func (r *removeCategoryParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = removeCategoryParam(v)

	return r.validate()
}

func (r *removeCategoryParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("invalid category id")
	}

	return nil
}
