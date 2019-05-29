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

type LogType string

const (
	LogTypeUser    LogType = "USER"
	LogTypeGroup   LogType = "GROUP"
	LogTypeSwitch  LogType = "SWITCH"
	LogTypeNetwork LogType = "NETWORK"
	LogTypeHost    LogType = "HOST"
	LogTypeVIP     LogType = "VIP"
)

func (r LogType) Validate() error {
	if r != LogTypeUser && r != LogTypeGroup && r != LogTypeSwitch &&
		r != LogTypeNetwork && r != LogTypeHost && r != LogTypeVIP {
		return fmt.Errorf("invalid log type: %v", r)
	}

	return nil
}

type LogMethod string

const (
	LogMethodAdd    LogMethod = "ADD"
	LogMethodUpdate LogMethod = "UPDATE"
	LogMethodRemove LogMethod = "REMOVE"
)

func (r LogMethod) Validate() error {
	if r != LogMethodAdd && r != LogMethodUpdate && r != LogMethodRemove {
		return fmt.Errorf("invalid log method: %v", r)
	}

	return nil
}

type LogTransaction interface {
	// Logs returns a list of registered logs. Search can be nil that means no search.
	QueryLog(*Search, Pagination) ([]*Log, error)
}

type Log struct {
	ID        uint64
	User      string
	Type      LogType
	Method    LogMethod
	Data      string
	Timestamp time.Time
}

func (r *Log) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID        uint64 `json:"id"`
		User      string `json:"user"`
		Type      string `json:"type"`
		Method    string `json:"method"`
		Data      string `json:"data"`
		Timestamp int64  `json:"timestamp"`
	}{
		ID:        r.ID,
		User:      r.User,
		Type:      string(r.Type),
		Method:    string(r.Method),
		Data:      r.Data,
		Timestamp: r.Timestamp.Unix(),
	})
}

func (r *API) listLog(w api.ResponseWriter, req *rest.Request) {
	p := new(listLogParam)
	if err := req.DecodeJsonPayload(p); err != nil {
		w.Write(api.Response{Status: api.StatusInvalidParameter, Message: fmt.Sprintf("failed to decode param: %v", err.Error())})
		return
	}
	logger.Debugf("listLog request from %v: %v", req.RemoteAddr, spew.Sdump(p))

	if _, ok := r.session.Get(p.SessionID); ok == false {
		w.Write(api.Response{Status: api.StatusUnknownSession, Message: fmt.Sprintf("unknown session id: %v", p.SessionID)})
		return
	}

	var log []*Log
	f := func(tx Transaction) (err error) {
		log, err = tx.QueryLog(p.Search, p.Pagination)
		return err
	}
	if err := r.DB.Exec(f); err != nil {
		w.Write(api.Response{Status: api.StatusInternalServerError, Message: fmt.Sprintf("failed to query the log list: %v", err.Error())})
		return
	}
	logger.Debugf("queried log list: %v", spew.Sdump(log))

	w.Write(api.Response{Status: api.StatusOkay, Data: log})
}

type listLogParam struct {
	SessionID  string
	Search     *Search
	Pagination Pagination
}

func (r *listLogParam) UnmarshalJSON(data []byte) error {
	v := struct {
		SessionID  string     `json:"session_id"`
		Search     *Search    `json:"search"`
		Pagination Pagination `json:"pagination"`
	}{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = listLogParam(v)

	return r.validate()
}

func (r *listLogParam) validate() error {
	if len(r.SessionID) != 64 {
		return errors.New("invalid session id")
	}
	// If search is nil, fetch logs without using search.
	if r.Search != nil {
		if r.Search.Key <= ColumnDescription || r.Search.Key > ColumnLogMethod {
			return errors.New("invalid search key")
		}
		if err := r.Search.Validate(); err != nil {
			return err
		}
	}
	if r.Pagination.Limit == 0 {
		return errors.New("invalid pagination limit")
	}

	return nil
}
