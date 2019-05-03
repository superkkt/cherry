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

type UserTransaction interface {
	// Auth returns information for a user if name and password match. Otherwise, it returns nil.
	Auth(name, password string) (*User, error)
	Users(Pagination) ([]*User, error)
	AddUser(name, password string) (user *User, duplicated bool, err error)
	// UpdateUser updates password and admin authorization of a user specified by id and then returns information of the user. It returns nil if the user does not exist.
	UpdateUser(id uint64, password *string, admin *bool) (*User, error)
	// ActivateUser enables a user specified by id and then returns information of the user. It returns nil if the user does not exist.
	ActivateUser(id uint64) (*User, error)
	// DeactivateUser disables a user specified by id and then returns information of the user. It returns nil if the user does not exist.
	DeactivateUser(id uint64) (*User, error)
}

type User struct {
	ID        uint64
	Name      string
	Enabled   bool
	Admin     bool
	Timestamp time.Time
}

func (r *User) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID        uint64 `json:"id"`
		Name      string `json:"name"`
		Enabled   bool   `json:"enabled"`
		Admin     bool   `json:"admin"`
		Timestamp int64  `json:"timestamp"`
	}{
		ID:        r.ID,
		Name:      r.Name,
		Enabled:   r.Enabled,
		Admin:     r.Admin,
		Timestamp: r.Timestamp.Unix(),
	})
}

func (r *API) login(w api.ResponseWriter, req *rest.Request) {
	p := new(loginParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("login request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	var user *User
	f := func(tx Transaction) (err error) {
		user, err = tx.Auth(p.Name, p.Password)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to authenticate an user account: %v", err.Error())})
		return
	}

	if user == nil {
		w.Write(api.Response{Status: api.StatusIncorrectCredential, Message: fmt.Sprintf("incorrect username or password: username=%v", p.Name)})
		return
	}
	if user.Enabled == false {
		w.Write(api.Response{Status: api.StatusBlockedAccount, Message: fmt.Sprintf("login attempt with a blocked account: %v", p.Name)})
		return
	}

	id := r.session.Add(user)
	logger.Debugf("login success: user=%v, sessionID=%v", spew.Sdump(user), id)

	w.Write(api.Response{
		Status: api.StatusOkay,
		Data: struct {
			SessionID string `json:"session_id"`
			ID        uint64 `json:"id"`
			Admin     bool   `json:"admin"`
		}{
			SessionID: id,
			ID:        user.ID,
			Admin:     user.Admin,
		},
	})
}

type loginParam struct {
	Name     string
	Password string
}

func (r *loginParam) UnmarshalJSON(data []byte) error {
	v := struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = loginParam(v)

	return r.validate()
}

func (r *loginParam) validate() error {
	if len(r.Name) < 3 || len(r.Name) > 24 {
		return fmt.Errorf("invalid name: %v", r.Name)
	}
	if len(r.Password) < 8 || len(r.Password) > 64 {
		return fmt.Errorf("invalid password: %v", r.Password)
	}

	return nil
}

func (r *API) logout(w api.ResponseWriter, req *rest.Request) {
	p := new(logoutParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("logout request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if r.session.Remove(p.SessionID) == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("logout attempt with an unknown session ID: %v", p.SessionID)})
		return
	}
	logger.Debugf("session removed: sessionID=%v", p.SessionID)

	w.Write(api.Response{Status: api.StatusOkay})
}

type logoutParam struct {
	SessionID string
}

func (r *logoutParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = logoutParam(v)

	return r.validate()
}

func (r *logoutParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}

	return nil
}

func (r *API) listUser(w api.ResponseWriter, req *rest.Request) {
	p := new(listUserParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("listUser request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if r.validateAdminSession(p.SessionID) == false {
		w.Write(api.Response{Status: api.StatusPermissionDenied, Message: fmt.Sprintf("not allowed admin session id: %v", p.SessionID)})
		return
	}

	var user []*User
	f := func(tx Transaction) (err error) {
		user, err = tx.Users(p.Pagination)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to query the user accounts: %v", err.Error())})
		return
	}
	logger.Debugf("queried user accounts: %v", spew.Sdump(user))

	w.Write(api.Response{Status: api.StatusOkay, Data: user})
}

type listUserParam struct {
	SessionID  string
	Pagination Pagination
}

func (r *listUserParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID  string     `json:"session_id"`
		Pagination Pagination `json:"pagination"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = listUserParam(v)

	return r.validate()
}

func (r *listUserParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.Pagination.Limit == 0 {
		return errors.New("invalid pagination limit")
	}

	return nil
}

func (r *API) addUser(w api.ResponseWriter, req *rest.Request) {
	p := new(addUserParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode params: %v", err.Error())})
		return
	}
	logger.Debugf("addUser request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if r.validateAdminSession(p.SessionID) == false {
		w.Write(api.Response{Status: api.StatusPermissionDenied, Message: fmt.Sprintf("not allowed admin session id: %v", p.SessionID)})
		return
	}

	var user *User
	var duplicated bool
	f := func(tx Transaction) (err error) {
		user, duplicated, err = tx.AddUser(p.Name, p.Password)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to add a new user account: %v", err.Error())})
		return
	}

	if duplicated {
		w.Write(api.Response{Status: api.StatusDuplicated, Message: fmt.Sprintf("duplicated user account: %v", p.Name)})
		return
	}
	logger.Debugf("added the user account: %v", spew.Sdump(user))

	w.Write(api.Response{Status: api.StatusOkay, Data: user})
}

type addUserParam struct {
	SessionID string
	Name      string
	Password  string
}

func (r *addUserParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		Name      string `json:"name"`
		Password  string `json:"password"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = addUserParam(v)

	return r.validate()
}

func (r *addUserParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if len(r.Name) < 3 || len(r.Name) > 24 {
		return fmt.Errorf("invalid name: %v", r.Name)
	}
	if len(r.Password) < 8 || len(r.Password) > 64 {
		return fmt.Errorf("invalid password: %v", r.Password)
	}

	return nil
}

func (r *API) updateUser(w api.ResponseWriter, req *rest.Request) {
	p := new(updateUserParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode params: %v", err.Error())})
		return
	}
	logger.Debugf("updateUser request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	s, ok := r.session.Get(p.SessionID)
	if ok == false || (s.(*User).Admin == false && s.(*User).ID != p.ID) {
		w.Write(api.Response{Status: api.StatusPermissionDenied, Message: fmt.Sprintf("not allowed session id: %v", p.SessionID)})
		return
	}

	// Non-admin users can modify only password.
	if s.(*User).Admin == false {
		p.Admin = nil
	}

	var user *User
	f := func(tx Transaction) (err error) {
		user, err = tx.UpdateUser(p.ID, p.Password, p.Admin)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to update a user account: %v", err.Error())})
		return
	}

	if user == nil {
		w.Write(api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found user to update: %v", p.ID)})
		return
	}
	logger.Debugf("updated the user account: %v", spew.Sdump(user))

	w.Write(api.Response{Status: api.StatusOkay, Data: user})
}

type updateUserParam struct {
	SessionID string
	ID        uint64
	Password  *string
	Admin     *bool
}

func (r *updateUserParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string  `json:"session_id"`
		ID        uint64  `json:"id"`
		Password  *string `json:"password"`
		Admin     *bool   `json:"admin"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = updateUserParam(v)

	return r.validate()
}

func (r *updateUserParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("invalid user ID")
	}
	if r.Password == nil && r.Admin == nil {
		return errors.New("empty parameter")
	}
	if r.Password != nil && (len(*r.Password) < 8 || len(*r.Password) > 64) {
		return fmt.Errorf("invalid password: %v", *r.Password)
	}

	return nil
}

func (r *API) activateUser(w api.ResponseWriter, req *rest.Request) {
	p := new(activateUserParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode params: %v", err.Error())})
		return
	}
	logger.Debugf("activateUser request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if r.validateAdminSession(p.SessionID) == false {
		w.Write(api.Response{Status: api.StatusPermissionDenied, Message: fmt.Sprintf("not allowed admin session id: %v", p.SessionID)})
		return
	}

	var user *User
	f := func(tx Transaction) (err error) {
		user, err = tx.ActivateUser(p.ID)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to activate an user account: %v", err.Error())})
		return
	}

	if user == nil {
		w.Write(api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found user to activate: %v", p.ID)})
		return
	}
	logger.Debugf("activated the user account: %v", spew.Sdump(user))

	w.Write(api.Response{Status: api.StatusOkay})
}

type activateUserParam struct {
	SessionID string
	ID        uint64
}

func (r *activateUserParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = activateUserParam(v)

	return r.validate()
}

func (r *activateUserParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("invalid user id")
	}

	return nil
}

func (r *API) deactivateUser(w api.ResponseWriter, req *rest.Request) {
	p := new(deactivateUserParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode params: %v", err.Error())})
		return
	}
	logger.Debugf("deactivateUser request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if r.validateAdminSession(p.SessionID) == false {
		w.Write(api.Response{Status: api.StatusPermissionDenied, Message: fmt.Sprintf("not allowed admin session id: %v", p.SessionID)})
		return
	}

	var user *User
	f := func(tx Transaction) (err error) {
		user, err = tx.DeactivateUser(p.ID)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to deactivate an user account: %v", err.Error())})
		return
	}

	if user == nil {
		w.Write(api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found user to deactivate: %v", p.ID)})
		return
	}
	logger.Debugf("deactivated the user account: %v", spew.Sdump(user))

	w.Write(api.Response{Status: api.StatusOkay})
}

type deactivateUserParam struct {
	SessionID string
	ID        uint64
}

func (r *deactivateUserParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = deactivateUserParam(v)

	return r.validate()
}

func (r *deactivateUserParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("invalid user id")
	}

	return nil
}
