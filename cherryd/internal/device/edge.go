/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package device

import (
	"fmt"
	"git.sds.co.kr/cherry.git/cherryd/internal/graph"
)

type Point struct {
	Node *Device
	Port uint32
}

func (r Point) ID() string {
	return fmt.Sprintf("%v:%v", r.Node.DPID, r.Port)
}

func (r Point) Vertex() graph.Vertex {
	return r.Node
}

func (r Point) Compare(p Point) bool {
	return r.Node.DPID == p.Node.DPID && r.Port == p.Port
}

type Edge struct {
	P1, P2 *Point
	weight float64
}

func newEdge(p1, p2 *Point, weight float64) *Edge {
	if p1 == nil || p2 == nil {
		panic("nil point")
	}

	return &Edge{
		P1:     p1,
		P2:     p2,
		weight: weight,
	}
}

func (r Edge) ID() string {
	first := r.P1
	second := r.P2
	if first.Node.DPID > second.Node.DPID {
		first, second = second, first
	}

	return fmt.Sprintf("%v:%v/%v:%v", first.Node.DPID, first.Port, second.Node.DPID, second.Port)
}

func (r Edge) Points() [2]graph.Point {
	return [2]graph.Point{r.P1, r.P2}
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
