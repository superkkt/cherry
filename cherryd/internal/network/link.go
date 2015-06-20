/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
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

package network

import (
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/graph"
	"sort"
)

type link struct {
	ports [2]*Port
}

func newLink(ports [2]*Port) *link {
	return &link{
		ports: ports,
	}
}

func (r *link) ID() string {
	s := []string{r.ports[0].ID(), r.ports[1].ID()}
	sort.Strings(s)

	return fmt.Sprintf("%v/%v", s[0], s[1])
}

func (r *link) Points() [2]graph.Point {
	return [2]graph.Point{r.ports[0], r.ports[1]}
}

func (r *link) Weight() float64 {
	// TODO: Calculate weight dynamically based on the link speed among these two ports
	return 0
}
