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
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/davecgh/go-spew/spew"
)

/*
 * Status Codes:
 *
 * 200 = Okay.
 * 4xx = Client-side errors.
 * 5xx = Server-side errors.
 */
type Status int

const (
	StatusOkay = 200

	StatusInvalidParameter    = 400
	StatusIncorrectCredential = 401
	StatusUnknownSession      = 402
	StatusPermissionDenied    = 403
	StatusDuplicated          = 404
	StatusNotFound            = 405
	StatusBlockedAccount      = 406
	StatusBlockedHost         = 407

	StatusInternalServerError = 500
	StatusServiceUnavailable  = 501
)

type Response struct {
	Status  Status      `json:"status"`
	Message string      `json:"message,omitempty"` // Human readable message related with the status code.
	Data    interface{} `json:"data,omitempty"`
}

func ResponseHandler(f func(ResponseWriter, *rest.Request)) func(rest.ResponseWriter, *rest.Request) {
	return func(w rest.ResponseWriter, req *rest.Request) {
		lw := &logWriter{w: w}
		f(lw, req)
	}
}

type ResponseWriter interface {
	// Identical to the http.ResponseWriter interface
	Header() http.Header

	Write(Response)

	// Similar to the http.ResponseWriter interface, with additional JSON related
	// headers set.
	WriteHeader(int)
}

type logWriter struct {
	w rest.ResponseWriter
}

func (r *logWriter) Header() http.Header {
	return r.w.Header()
}

func (r *logWriter) Write(resp Response) {
	switch {
	case resp.Status >= StatusInternalServerError:
		logger.Errorf("server-side error response: status=%v, message=%v", resp.Status, resp.Message)
	case resp.Status >= StatusInvalidParameter:
		logger.Infof("client-side error response: status=%v, message=%v", resp.Status, resp.Message)
	default:
		logger.Debugf("success response: %v", spew.Sdump(resp))
	}

	if err := r.w.WriteJson(resp); err != nil {
		logger.Errorf("failed to write a JSON response: %v", err)
	}
}

func (r *logWriter) WriteHeader(status int) {
	r.w.WriteHeader(status)
}
