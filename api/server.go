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
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
)

type Server struct {
	Port uint16
	TLS  struct {
		Cert string // Path for a TLS certification file.
		Key  string // Path for a TLS private key file.
	}
	Observer   Observer
	Controller Controller
}

type Observer interface {
	IsMaster() bool
}

type Controller interface {
	Announce(net.IP, net.HardwareAddr) error
	RemoveFlows() error
	RemoveFlowsByMAC(net.HardwareAddr) error
}

func (r *Server) validate() error {
	if r.Observer == nil {
		return errors.New("nil observer")
	}
	if r.Controller == nil {
		return errors.New("nil controller")
	}

	return nil
}

func (r *Server) Serve(routes ...*rest.Route) error {
	if err := r.validate(); err != nil {
		return err
	}

	api := rest.NewApi()
	// Middleware to deny the client requests if we are not the master controller.
	api.Use(rest.MiddlewareSimple(func(handler rest.HandlerFunc) rest.HandlerFunc {
		return func(writer rest.ResponseWriter, request *rest.Request) {
			if r.Observer.IsMaster() == false {
				writer.WriteJson(Response{Status: StatusServiceUnavailable, Message: "use the master controller server"})
				return
			}
			handler(writer, request)
		}
	}))
	// Middleware to set the CORS header.
	api.Use(rest.MiddlewareSimple(func(handler rest.HandlerFunc) rest.HandlerFunc {
		return func(writer rest.ResponseWriter, request *rest.Request) {
			writer.Header().Set("Access-Control-Allow-Origin", "*")
			handler(writer, request)
		}
	}))
	router, err := rest.MakeRouter(routes...)
	if err != nil {
		return err
	}
	api.SetApp(router)

	// Listen on all interfaces.
	addr := fmt.Sprintf(":%v", r.Port)
	if r.TLS.Cert != "" && r.TLS.Key != "" {
		err = http.ListenAndServeTLS(addr, r.TLS.Cert, r.TLS.Key, api.MakeHandler())
	} else {
		err = http.ListenAndServe(addr, api.MakeHandler())
	}

	return err
}
