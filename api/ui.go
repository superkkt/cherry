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

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/davecgh/go-spew/spew"
)

type UI struct {
	Config
	DB Database

	session *session
}

type Database interface {
	// Auth returns information for a user if name and password match. Otherwise, it returns nil.
	Auth(name, password string) (*User, error)
	Users(offset uint32, limit uint8) ([]User, error)
	AddUser(name, password string) (id uint64, duplicated bool, err error)
	UpdateUser(id uint64, password *string, admin *bool) error
	ActivateUser(id uint64) error
	DeactivateUser(id uint64) error

	Groups(offset uint32, limit uint8) ([]Group, error)
	AddGroup(name string) (id uint64, duplicated bool, err error)
	UpdateGroup(id uint64, name string) (duplicated bool, err error)
	RemoveGroup(id uint64) error
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

func (r *UI) Serve() error {
	if r.DB == nil {
		return errors.New("nil DB")
	}
	r.session = newSession(256, 2*time.Hour)

	return r.serve(
		rest.Post("/api/v1/user/login", r.login),
		rest.Post("/api/v1/user/logout", r.logout),
		rest.Post("/api/v1/user/list", r.listUser),
		rest.Post("/api/v1/user/add", r.addUser),
		rest.Post("/api/v1/user/update", r.updateUser),
		rest.Post("/api/v1/user/activate", r.activateUser),
		rest.Post("/api/v1/user/deactivate", r.deactivateUser),
		rest.Post("/api/v1/group/list", r.listGroup),
		rest.Post("/api/v1/group/add", r.addGroup),
		rest.Post("/api/v1/group/update", r.updateGroup),
		rest.Post("/api/v1/group/remove", r.removeGroup),
	)
}

func (r *UI) login(w rest.ResponseWriter, req *rest.Request) {
	p := new(loginParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("login request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	user, err := r.DB.Auth(p.Name, p.Password)
	if err != nil {
		logger.Errorf("failed to authenticate an user account: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	if user == nil {
		logger.Infof("incorrect credential: user=%v, password=%v", p.Name, p.Password)
		w.WriteJson(&response{Status: statusIncorrectCredential, Message: "incorrect username or password"})
		return
	}
	if user.Enabled == false {
		logger.Infof("login attempt with a blocked account: user=%v", p.Name)
		w.WriteJson(&response{Status: statusBlockedAccount, Message: fmt.Sprintf("blocked account: %v", p.Name)})
		return
	}

	id := r.session.Add(user)
	logger.Debugf("login success: user=%v, sessionID=%v", spew.Sdump(user), id)

	w.WriteJson(&response{
		Status: statusOkay,
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

func (r *UI) logout(w rest.ResponseWriter, req *rest.Request) {
	p := new(logoutParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("logout request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if r.session.Remove(p.SessionID) == false {
		logger.Infof("logout attempt with an unknown session ID: %v", p.SessionID)
		w.WriteJson(&response{Status: statusUnknownSession, Message: fmt.Sprintf("unknown session ID: %v", p.SessionID)})
		return
	}
	logger.Debugf("session removed: sessionID=%v", p.SessionID)

	w.WriteJson(&response{Status: statusOkay})
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

func (r *UI) listUser(w rest.ResponseWriter, req *rest.Request) {
	p := new(listUserParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("listUser request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if r.validateAdminSession(p.SessionID) == false {
		logger.Warningf("invalid (or not allowed) admin session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusPermissionDenied, Message: fmt.Sprintf("not allowed admin session id: %v", p.SessionID)})
		return
	}

	user, err := r.DB.Users(p.Offset, p.Limit)
	if err != nil {
		logger.Errorf("failed to query the user list: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	logger.Debugf("queried user list: %v", spew.Sdump(user))

	w.WriteJson(&response{
		Status: statusOkay,
		Data:   user,
	})
}

type listUserParam struct {
	SessionID string
	Offset    uint32
	Limit     uint8
}

func (r *listUserParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		Offset    uint32 `json:"offset"`
		Limit     uint8  `json:"limit"`
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
	if r.Limit == 0 {
		return errors.New("invalid limit")
	}

	return nil
}

func (r *UI) addUser(w rest.ResponseWriter, req *rest.Request) {
	p := new(addUserParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("addUser request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if r.validateAdminSession(p.SessionID) == false {
		logger.Warningf("invalid (or not allowed) admin session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusPermissionDenied, Message: fmt.Sprintf("not allowed admin session id: %v", p.SessionID)})
		return
	}

	id, duplicated, err := r.DB.AddUser(p.Name, p.Password)
	if err != nil {
		logger.Errorf("failed to add a new user: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	if duplicated {
		logger.Infof("duplicated user account: name=%v", p.Name)
		w.WriteJson(&response{Status: statusDuplicated, Message: fmt.Sprintf("duplicated user account: %v", p.Name)})
		return
	}
	logger.Debugf("added user info: %v", spew.Sdump(p))

	w.WriteJson(&response{
		Status: statusOkay,
		Data: struct {
			ID uint64 `json:"id"`
		}{id},
	})
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

func (r *UI) updateUser(w rest.ResponseWriter, req *rest.Request) {
	p := new(updateUserParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("updateUser request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	s, ok := r.session.Get(p.SessionID)
	if ok == false || (s.(*User).Admin == false && s.(*User).ID != p.ID) {
		logger.Warningf("invalid (or not allowed) session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusPermissionDenied, Message: fmt.Sprintf("not allowed session id: %v", p.SessionID)})
		return
	}

	// Non-admin users can modify only password.
	if s.(*User).Admin == false {
		p.Admin = nil
	}

	if err := r.DB.UpdateUser(p.ID, p.Password, p.Admin); err != nil {
		logger.Errorf("failed to update user info: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	logger.Debugf("updated user info: %v", spew.Sdump(p))

	w.WriteJson(&response{Status: statusOkay})
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

func (r *UI) activateUser(w rest.ResponseWriter, req *rest.Request) {
	p := new(activateUserParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("activateUser request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if r.validateAdminSession(p.SessionID) == false {
		logger.Warningf("invalid (or not allowed) admin session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusPermissionDenied, Message: fmt.Sprintf("not allowed admin session id: %v", p.SessionID)})
		return
	}

	if err := r.DB.ActivateUser(p.ID); err != nil {
		logger.Errorf("failed to activate an user account: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	logger.Debugf("activated an user account: ID=%v", p.ID)

	w.WriteJson(&response{Status: statusOkay})
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

func (r *UI) deactivateUser(w rest.ResponseWriter, req *rest.Request) {
	p := new(deactivateUserParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("deactivateUser request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if r.validateAdminSession(p.SessionID) == false {
		logger.Warningf("invalid (or not allowed) admin session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusPermissionDenied, Message: fmt.Sprintf("not allowed admin session id: %v", p.SessionID)})
		return
	}

	if err := r.DB.DeactivateUser(p.ID); err != nil {
		logger.Errorf("failed to deactivate an user account: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	logger.Debugf("deactivated an user account: ID=%v", p.ID)

	w.WriteJson(&response{Status: statusOkay})
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

func (r *UI) validateAdminSession(sessionID string) bool {
	session, ok := r.session.Get(sessionID)
	if ok == false {
		return false
	}

	return session.(*User).Admin
}

func (r *UI) listGroup(w rest.ResponseWriter, req *rest.Request) {
	p := new(listGroupParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("listGroup request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	group, err := r.DB.Groups(p.Offset, p.Limit)
	if err != nil {
		logger.Errorf("failed to query the group list: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	logger.Debugf("queried group list: %v", spew.Sdump(group))

	w.WriteJson(&response{
		Status: statusOkay,
		Data:   group,
	})
}

type listGroupParam struct {
	SessionID string
	Offset    uint32
	Limit     uint8
}

func (r *listGroupParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		Offset    uint32 `json:"offset"`
		Limit     uint8  `json:"limit"`
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
	if r.Limit == 0 {
		return errors.New("invalid limit")
	}

	return nil
}

func (r *UI) addGroup(w rest.ResponseWriter, req *rest.Request) {
	p := new(addGroupParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("addGroup request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	id, duplicated, err := r.DB.AddGroup(p.Name)
	if err != nil {
		logger.Errorf("failed to add a new group: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	if duplicated {
		logger.Infof("duplicated group: name=%v", p.Name)
		w.WriteJson(&response{Status: statusDuplicated, Message: fmt.Sprintf("duplicated group: %v", p.Name)})
		return
	}
	logger.Debugf("added group info: %v", spew.Sdump(p))

	w.WriteJson(&response{
		Status: statusOkay,
		Data: &struct {
			ID uint64 `json:"id"`
		}{id},
	})
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

func (r *UI) updateGroup(w rest.ResponseWriter, req *rest.Request) {
	p := new(updateGroupParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("updateGroup request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	duplicated, err := r.DB.UpdateGroup(p.ID, p.Name)
	if err != nil {
		logger.Errorf("failed to update group info: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	if duplicated {
		logger.Infof("duplicated group: name=%v", p.Name)
		w.WriteJson(&response{Status: statusDuplicated, Message: fmt.Sprintf("duplicated group: %v", p.Name)})
		return
	}
	logger.Debugf("updated group info: %v", spew.Sdump(p))

	w.WriteJson(&response{Status: statusOkay})
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

func (r *UI) removeGroup(w rest.ResponseWriter, req *rest.Request) {
	p := new(removeGroupParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("removeGroup request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	if err := r.DB.RemoveGroup(p.ID); err != nil {
		logger.Errorf("failed to remove group info: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	logger.Debugf("removed group info: %v", spew.Sdump(p))

	w.WriteJson(&response{Status: statusOkay})
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
