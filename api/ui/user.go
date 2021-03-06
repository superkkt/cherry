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
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

type UserTransaction interface {
	User(name string) (*User, error)
	Users(Pagination) ([]*User, error)
	AddUser(requesterID uint64, name, key string) (user *User, duplicated bool, err error)
	// UpdateUser updates enabled and admin authorization of a user specified by id and then returns information of the user. It returns nil if the user does not exist.
	UpdateUser(requesterID, userID uint64, enabled, admin *bool) (*User, error)
	ResetOTPKey(name, key string) (ok bool, err error)
}

type User struct {
	ID        uint64
	Name      string
	Key       string // Key used in OTP Authentication.
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
	logger.Debugf("login request from %v: %v", req.RemoteAddr, spew.Sdump(&struct {
		Name string `json:"name"`
		Code string `json:"code"`
	}{
		Name: p.Name,
		Code: p.Code,
	}))

	var user *User
	f := func(tx Transaction) (err error) {
		user, err = tx.User(p.Name)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to query an user account: %v", err.Error())})
		return
	}

	if user == nil {
		w.Write(api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found user to login: %v", p.Name)})
		return
	}
	if user.Enabled == false {
		w.Write(api.Response{Status: api.StatusBlockedAccount, Message: fmt.Sprintf("login attempt with a blocked account: %v", p.Name)})
		return
	}

	ok, err := r.LDAP.Auth(p.Name, p.Password)
	if err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to authenticate an user account: %v", err.Error())})
		return
	}
	if ok == false {
		w.Write(api.Response{Status: api.StatusIncorrectCredential, Message: fmt.Sprintf("incorrect username or password: %v", p.Name)})
		return
	}

	if ok := totp.Validate(p.Code, user.Key); ok == false {
		w.Write(api.Response{Status: api.StatusIncorrectCredential, Message: fmt.Sprintf("incorrect OTP code: %v", p.Code)})
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
	Code     string // OTP Authentication Code.
}

func (r *loginParam) UnmarshalJSON(data []byte) error {
	v := struct {
		Name     string `json:"name"`
		Password string `json:"password"`
		Code     string `json:"code"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = loginParam(v)

	return r.validate()
}

func (r *loginParam) validate() error {
	if len(r.Name) == 0 {
		return errors.New("empty name")
	}
	if len(r.Password) == 0 {
		return errors.New("empty password")
	}
	if len(r.Code) != 6 {
		return fmt.Errorf("invalid OTP code: %v", r.Code)
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

	session, ok := r.session.Get(p.SessionID)
	if ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}
	if session.(*User).Admin == false {
		w.Write(api.Response{Status: api.StatusPermissionDenied, Message: fmt.Sprintf("not allowed session id: %v", p.SessionID)})
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

	session, ok := r.session.Get(p.SessionID)
	if ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}
	if session.(*User).Admin == false {
		w.Write(api.Response{Status: api.StatusPermissionDenied, Message: fmt.Sprintf("not allowed session id: %v", p.SessionID)})
		return
	}

	key, err := generateOTPKey(p.Name)
	if err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to generate OTP key: %v", err)})
		return
	}

	var user *User
	var duplicated bool
	f := func(tx Transaction) (err error) {
		user, duplicated, err = tx.AddUser(session.(*User).ID, p.Name, key.Secret())
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

	w.Write(api.Response{
		Status: api.StatusOkay,
		Data: &struct {
			User *User  `json:"user"`
			OTP  string `json:"otp"`
		}{
			User: user,
			OTP:  key.String(),
		},
	})
}

type addUserParam struct {
	SessionID string
	Name      string
}

func (r *addUserParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		Name      string `json:"name"`
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
	if len(r.Name) == 0 {
		return errors.New("empty name")
	}

	return nil
}

func generateOTPKey(account string) (*otp.Key, error) {
	return totp.Generate(totp.GenerateOpts{
		Issuer:      "Cherry",
		AccountName: account,
	})
}

func (r *API) updateUser(w api.ResponseWriter, req *rest.Request) {
	p := new(updateUserParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode params: %v", err.Error())})
		return
	}
	logger.Debugf("updateUser request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	session, ok := r.session.Get(p.SessionID)
	if ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}
	if session.(*User).Admin == false {
		w.Write(api.Response{Status: api.StatusPermissionDenied, Message: fmt.Sprintf("not allowed session id: %v", p.SessionID)})
		return
	}

	var user *User
	f := func(tx Transaction) (err error) {
		user, err = tx.UpdateUser(session.(*User).ID, p.ID, p.Enabled, p.Admin)
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
	Enabled   *bool
	Admin     *bool
}

func (r *updateUserParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
		Enabled   *bool  `json:"enabled"`
		Admin     *bool  `json:"admin"`
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
	if r.Enabled == nil && r.Admin == nil {
		return errors.New("empty parameter")
	}

	return nil
}

func (r *API) resetOTP(w api.ResponseWriter, req *rest.Request) {
	p := new(resetOTPParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode params: %v", err.Error())})
		return
	}
	logger.Debugf("resetOTP request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	session, ok := r.session.Get(p.SessionID)
	if ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}
	if session.(*User).Admin == false {
		w.Write(api.Response{Status: api.StatusPermissionDenied, Message: fmt.Sprintf("not allowed session id: %v", p.SessionID)})
		return
	}

	key, err := generateOTPKey(p.Name)
	if err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to generate OTP key: %v", err)})
		return
	}

	f := func(tx Transaction) (err error) {
		ok, err = tx.ResetOTPKey(p.Name, key.Secret())
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to reset OTP of a user account: %v", err.Error())})
		return
	}

	if ok == false {
		w.Write(api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found user to reset OTP: %v", p.Name)})
		return
	}
	logger.Debugf("reset OTP of the user account: %v", p.Name)

	w.Write(api.Response{Status: api.StatusOkay, Data: key.String()})
}

type resetOTPParam struct {
	SessionID string
	Name      string
}

func (r *resetOTPParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		Name      string `json:"name"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = resetOTPParam(v)

	return r.validate()
}

func (r *resetOTPParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if len(r.Name) == 0 {
		return errors.New("empty name")
	}

	return nil
}
