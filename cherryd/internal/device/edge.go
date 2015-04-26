/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package device

import (
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/graph"
)

type Vertex struct {
	Node *Device
	Port uint32
}

func (r Vertex) ID() string {
	if r.Node == nil {
		panic("nil vertex node")
	}
	return r.Node.ID()
}

type Edge struct {
	v1, v2 *Vertex
	weight float64
}

func newEdge(v1, v2 *Vertex, weight float64) *Edge {
	if v1 == nil || v2 == nil {
		panic("nil vertex")
	}

	return &Edge{
		v1:     v1,
		v2:     v2,
		weight: weight,
	}
}

func (r Edge) ID() string {
	first := r.v1
	second := r.v2
	if first.Node.DPID > second.Node.DPID {
		first, second = second, first
	}

	return fmt.Sprintf("%v:%v/%v:%v", first.Node.DPID, first.Port, second.Node.DPID, second.Port)
}

func (r Edge) Vertexies() [2]graph.Vertex {
	return [2]graph.Vertex{r.v1, r.v2}
}

func (r Edge) Weight() float64 {
	return r.weight
}

func calculateEdgeWeight(speed uint64) float64 {
	// http://en.wikipedia.org/wiki/Spanning_Tree_Protocol#Data_rate_and_STP_path_cost
	switch speed {
	case 5:
		return 250.0
	case 10:
		return 100.0
	case 50:
		return 50.0
	case 100:
		return 19.0
	case 500:
		return 10.0
	case 1000:
		return 4.0
	case 10000:
		return 2.0
	case 40000:
		return 1.0
	case 100000:
		return 0.5
	case 1000000:
		return 0.25
	default:
		return 19.0 // fallback
	}
}
