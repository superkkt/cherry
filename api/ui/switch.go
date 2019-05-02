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
	"regexp"
	"strconv"
	"strings"

	"github.com/superkkt/cherry/api"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/davecgh/go-spew/spew"
)

type SwitchTransaction interface {
	Switches(offset uint32, limit uint8) ([]*Switch, error)
	AddSwitch(dpid uint64, nPorts, firstPort, firstPrintedPort uint16, desc string) (sw *Switch, duplicated bool, err error)
	// RemoveSwitch removes a switch specified by id and then returns information of the switch before removing. It returns nil if the switch does not exist.
	RemoveSwitch(id uint64) (*Switch, error)
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

func (r *API) listSwitch(w rest.ResponseWriter, req *rest.Request) {
	p := new(listSwitchParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&api.Response{Status: api.StatusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("listSwitch request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var sw []*Switch
	f := func(tx Transaction) (err error) {
		sw, err = tx.Switches(p.Offset, p.Limit)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.WriteJson(&api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to query the switch list: %v", err.Error())})
		return
	}
	logger.Debugf("queried switch list: %v", spew.Sdump(sw))

	w.WriteJson(&api.Response{Status: api.StatusOkay, Data: sw})
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

func (r *API) addSwitch(w rest.ResponseWriter, req *rest.Request) {
	p := new(addSwitchParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&api.Response{Status: api.StatusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("addSwitch request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var sw *Switch
	var duplicated bool
	f := func(tx Transaction) (err error) {
		sw, duplicated, err = tx.AddSwitch(p.DPID, p.NumPorts, p.FirstPort, p.FirstPrintedPort, p.Description)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.WriteJson(&api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to add a new switch: %v", err.Error())})
		return
	}

	if duplicated {
		logger.Infof("duplicated switch: dpid=%v", p.DPID)
		w.WriteJson(&api.Response{Status: api.StatusDuplicated, Message: fmt.Sprintf("duplicated switch: dpid=%v", p.DPID)})
		return
	}
	logger.Debugf("added switch info: %v", spew.Sdump(sw))

	w.WriteJson(&api.Response{Status: api.StatusOkay, Data: sw})
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

func (r *API) removeSwitch(w rest.ResponseWriter, req *rest.Request) {
	p := new(removeSwitchParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		logger.Warningf("failed to decode params: %v", err)
		w.WriteJson(&api.Response{Status: api.StatusInvalidParameter, Message: err.Error()})
		return
	}
	logger.Debugf("removeSwitch request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		logger.Warningf("unknown session id: %v", p.SessionID)
		w.WriteJson(&api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var sw *Switch
	f := func(tx Transaction) (err error) {
		sw, err = tx.RemoveSwitch(p.ID)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.WriteJson(&api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to remove a switch: %v", err.Error())})
		return
	}

	if sw == nil {
		logger.Infof("not found switch to remove: %v", p.ID)
		w.WriteJson(&api.Response{Status: api.StatusNotFound, Message: fmt.Sprintf("not found switch to remove: %v", p.ID)})
		return
	}
	logger.Debugf("removed a switch: %v", spew.Sdump(sw))

	logger.Debug("removing all flows from the entire switches")
	if err := r.Controller.RemoveFlows(); err != nil {
		// Ignore this error.
		logger.Errorf("failed to remove flows: %v", err)
	} else {
		logger.Debug("removed all flows from the entire switches")
	}

	w.WriteJson(&api.Response{Status: api.StatusOkay})
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
