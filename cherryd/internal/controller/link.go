/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package controller

import (
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/graph"
	"sort"
)

type Link struct {
	ports [2]*Port
}

func NewLink(ports [2]*Port) *Link {
	return &Link{
		ports: ports,
	}
}

func (r *Link) ID() string {
	s := []string{r.ports[0].ID(), r.ports[1].ID()}
	sort.Strings(s)

	return fmt.Sprintf("%v/%v", s[0], s[1])
}

func (r *Link) Points() [2]graph.Point {
	return [2]graph.Point{r.ports[0], r.ports[1]}
}

func (r *Link) Weight() float64 {
	// TODO: Calculate weight dynamically based on the link speed among these two ports
	return 0
}
