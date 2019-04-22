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

/*
 * Status Codes:
 *
 * 200 = Okay.
 * 4xx = Client-side errors.
 * 5xx = Server-side errors.
 */
type status int

const (
	statusOkay = 200

	statusInvalidParameterError = 400

	statusInternalServerError     = 500
	statusServiceUnavailableError = 501
)

type response struct {
	Status  status      `json:"status"`
	Message string      `json:"message,omitempty"` // Human readable message related with the status code.
	Data    interface{} `json:"data,omitempty"`
}
