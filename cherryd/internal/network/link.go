/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
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
