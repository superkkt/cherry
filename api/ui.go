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
	"net"
	"regexp"
	"strconv"
	"strings"
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

	Switches(offset uint32, limit uint8) ([]Switch, error)
	AddSwitch(dpid uint64, nPorts, firstPort, firstPrintedPort uint16, desc string) (id uint64, duplicated bool, err error)
	RemoveSwitch(id uint64) error

	Networks(offset uint32, limit uint8) ([]Network, error)
	AddNetwork(addr net.IP, mask net.IPMask) (id uint64, duplicated bool, err error)
	RemoveNetwork(id uint64) error
	IPAddrs(networkID uint64) ([]IP, error)
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

type Switch struct {
	ID               uint64 `json:"id"`
	DPID             uint64 `json:"dpid"`
	NumPorts         uint16 `json:"n_ports"`
	FirstPort        uint16 `json:"first_port"`
	FirstPrintedPort uint16 `json:"first_printed_port"`
	Description      string `json:"description"`
}

func (r *Switch) MarshalJSON() ([]byte, error) {
	s := new(struct {
		ID   uint64 `json:"id"`
		DPID struct {
			Int uint64 `json:"int"`
			Hex string `json:"hex"`
		} `json:"dpid"`
		NumPorts         uint16 `json:"n_ports"`
		FirstPort        uint16 `json:"first_port"`
		FirstPrintedPort uint16 `json:"first_printed_port"`
		Description      string `json:"description"`
	})

	s.ID = r.ID
	s.DPID.Int = r.DPID
	s.DPID.Hex = hexDPID(r.DPID)
	s.NumPorts = r.NumPorts
	s.FirstPort = r.FirstPort
	s.FirstPrintedPort = r.FirstPrintedPort
	s.Description = r.Description

	return json.Marshal(&s)
}

func hexDPID(dpid uint64) string {
	hex := fmt.Sprintf("%016x", dpid)
	re := regexp.MustCompile("..")
	return strings.TrimRight(re.ReplaceAllString(hex, "$0:"), ":")
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
		rest.Post("/api/v1/switch/list", r.listSwitch),
		rest.Post("/api/v1/switch/add", r.addSwitch),
		rest.Post("/api/v1/switch/remove", r.removeSwitch),
		rest.Post("/api/v1/network/list", r.listNetwork),
		rest.Post("/api/v1/network/add", r.addNetwork),
		rest.Post("/api/v1/network/remove", r.removeNetwork),
		rest.Post("/api/v1/network/ip", r.listIP),
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

	w.WriteJson(&response{Status: statusOkay, Data: user})
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

	w.WriteJson(&response{Status: statusOkay, Data: id})
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

	w.WriteJson(&response{Status: statusOkay, Data: group})
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

	w.WriteJson(&response{Status: statusOkay, Data: id})
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

func (r *UI) listSwitch(w rest.ResponseWriter, req *rest.Request) {
	p := new(listSwitchParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("listSwitch request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	sw, err := r.DB.Switches(p.Offset, p.Limit)
	if err != nil {
		logger.Errorf("failed to query the switch list: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	logger.Debugf("queried switch list: %v", spew.Sdump(sw))

	w.WriteJson(&response{Status: statusOkay, Data: sw})
}

type listSwitchParam struct {
	SessionID string
	Offset    uint32
	Limit     uint8
}

func (r *listSwitchParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		Offset    uint32 `json:"offset"`
		Limit     uint8  `json:"limit"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = listSwitchParam(v)

	return r.validate()
}

func (r *listSwitchParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.Limit == 0 {
		return errors.New("invalid limit")
	}

	return nil
}

func (r *UI) addSwitch(w rest.ResponseWriter, req *rest.Request) {
	p := new(addSwitchParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("addSwitch request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	id, duplicated, err := r.DB.AddSwitch(p.DPID, p.NumPorts, p.FirstPort, p.FirstPrintedPort, p.Description)
	if err != nil {
		logger.Errorf("failed to add a new switch: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	if duplicated {
		logger.Infof("duplicated switch: dpid=%v", p.DPID)
		w.WriteJson(&response{Status: statusDuplicated, Message: fmt.Sprintf("duplicated switch: dpid=%v", p.DPID)})
		return
	}
	logger.Debugf("added switch info: %v", spew.Sdump(p))

	w.WriteJson(&response{Status: statusOkay, Data: id})
}

type addSwitchParam struct {
	SessionID        string
	DPID             uint64
	NumPorts         uint16
	FirstPort        uint16
	FirstPrintedPort uint16
	Description      string
}

func (r *addSwitchParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID        string `json:"session_id"`
		DPID             string `json:"dpid"`
		NumPorts         uint16 `json:"n_ports"`
		FirstPort        uint16 `json:"first_port"`
		FirstPrintedPort uint16 `json:"first_printed_port"`
		Description      string `json:"description"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	if len(v.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if v.NumPorts == 0 {
		return errors.New("invalid number of ports")
	}
	if v.NumPorts > 512 {
		return errors.New("too many ports")
	}
	if len(v.Description) > 255 {
		return errors.New("too long description")
	}
	if uint32(v.FirstPort)+uint32(v.NumPorts) > 0xFFFF {
		return errors.New("too high first port number")
	}
	ok, err := regexp.MatchString("^([0-9a-fA-F]{2}:){7}([0-9a-fA-F]{2})$", v.DPID)
	if err != nil {
		return err
	}

	// Is the DP id in hex format?
	if ok {
		v.DPID = strings.Replace(v.DPID, ":", "", -1)
		if r.DPID, err = strconv.ParseUint(v.DPID, 16, 64); err != nil {
			return err
		}
	} else {
		if r.DPID, err = strconv.ParseUint(v.DPID, 10, 64); err != nil {
			return err
		}
	}

	r.SessionID = v.SessionID
	r.NumPorts = v.NumPorts
	r.FirstPort = v.FirstPort
	r.FirstPrintedPort = v.FirstPrintedPort
	r.Description = v.Description

	return nil
}

func (r *UI) removeSwitch(w rest.ResponseWriter, req *rest.Request) {
	p := new(removeSwitchParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("removeSwitch request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	if err := r.DB.RemoveSwitch(p.ID); err != nil {
		logger.Errorf("failed to remove switch info: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	logger.Debugf("removed switch info: %v", spew.Sdump(p))

	logger.Debug("removing all flows from the entire switches")
	if err := r.Controller.RemoveFlows(); err != nil {
		// Ignore this error.
		logger.Errorf("failed to remove flows: %v", err)
	}
	logger.Debug("removed all flows from the entire switches")

	w.WriteJson(&response{Status: statusOkay})
}

type removeSwitchParam struct {
	SessionID string
	ID        uint64
}

func (r *removeSwitchParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = removeSwitchParam(v)

	return r.validate()
}

func (r *removeSwitchParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("invalid switch id")
	}

	return nil
}

func (r *UI) listNetwork(w rest.ResponseWriter, req *rest.Request) {
	p := new(listNetworkParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("listNetwork request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	network, err := r.DB.Networks(p.Offset, p.Limit)
	if err != nil {
		logger.Errorf("failed to query the network list: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	logger.Debugf("queried network list: %v", spew.Sdump(network))

	w.WriteJson(&response{Status: statusOkay, Data: network})
}

type listNetworkParam struct {
	SessionID string
	Offset    uint32
	Limit     uint8
}

func (r *listNetworkParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		Offset    uint32 `json:"offset"`
		Limit     uint8  `json:"limit"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = listNetworkParam(v)

	return r.validate()
}

func (r *listNetworkParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.Limit == 0 {
		return errors.New("invalid limit")
	}

	return nil
}

func (r *UI) addNetwork(w rest.ResponseWriter, req *rest.Request) {
	p := new(addNetworkParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("addNetwork request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	id, duplicated, err := r.DB.AddNetwork(p.Address, p.Mask)
	if err != nil {
		logger.Errorf("failed to add a new network: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	if duplicated {
		logger.Infof("duplicated network: address=%v, mask=%v", p.Address, p.Mask)
		w.WriteJson(&response{Status: statusDuplicated, Message: fmt.Sprintf("duplicated network: address=%v, mask=%v", p.Address, p.Mask)})
		return
	}
	logger.Debugf("added network info: %v", spew.Sdump(p))

	w.WriteJson(&response{Status: statusOkay, Data: id})
}

type addNetworkParam struct {
	SessionID string
	Address   net.IP
	Mask      net.IPMask
}

func (r *addNetworkParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		Address   string `json:"address"`
		Mask      uint8  `json:"mask"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	if len(v.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	addr := net.ParseIP(v.Address)
	if addr == nil {
		return fmt.Errorf("invalid network address: %v", v.Address)
	}
	if v.Mask < 24 || v.Mask > 30 {
		return fmt.Errorf("invalid network mask: %v", v.Mask)
	}

	r.SessionID = v.SessionID
	r.Mask = net.CIDRMask(int(v.Mask), 32)
	r.Address = addr.Mask(r.Mask)

	return nil
}

func (r *UI) removeNetwork(w rest.ResponseWriter, req *rest.Request) {
	p := new(removeNetworkParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("removeNetwork request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	if err := r.DB.RemoveNetwork(p.ID); err != nil {
		logger.Errorf("failed to remove network info: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	logger.Debugf("removed network info: %v", spew.Sdump(p))

	logger.Debug("removing all flows from the entire switches")
	if err := r.Controller.RemoveFlows(); err != nil {
		// Ignore this error.
		logger.Errorf("failed to remove flows: %v", err)
	}
	logger.Debug("removed all flows from the entire switches")

	w.WriteJson(&response{Status: statusOkay})
}

type removeNetworkParam struct {
	SessionID string
	ID        uint64
}

func (r *removeNetworkParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID string `json:"session_id"`
		ID        uint64 `json:"id"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = removeNetworkParam(v)

	return r.validate()
}

func (r *removeNetworkParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	if r.ID == 0 {
		return errors.New("empty network id")
	}

	return nil
}

func (r *UI) listIP(w rest.ResponseWriter, req *rest.Request) {
	p := new(listIPParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&response{Status: statusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("listIP request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&response{Status: statusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	ip, err := r.DB.IPAddrs(p.NetworkID)
	if err != nil {
		logger.Errorf("failed to query the network ip list: %v", err)
		w.WriteJson(&response{Status: statusInternalServerError, Message: err.Error()})
		return
	}
	logger.Debugf("queried network ip list: %v", spew.Sdump(ip))

	w.WriteJson(&response{Status: statusOkay, Data: ip})
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
