/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package graph

import (
	"fmt"
	"testing"
)

type node struct {
	dpid string
}

func (r node) ID() string {
	return fmt.Sprintf("%v", r.dpid)
}

type point struct {
	dpid string
	port uint32
}

func (r point) ID() string {
	return fmt.Sprintf("%v:%v", r.dpid, r.port)
}

func (r point) Vertex() Vertex {
	return node{r.dpid}
}

type link struct {
	points [2]point
	weight float64
}

func (r link) ID() string {
	return fmt.Sprintf("%v:%v/%v:%v", r.points[0].dpid, r.points[0].port, r.points[1].dpid, r.points[1].port)
}

func (r link) Points() [2]Point {
	return [2]Point{r.points[0], r.points[1]}
}

func (r link) Weight() float64 {
	return r.weight
}

func printEnabledEdges(g *Graph) (int, float64) {
	count := 0
	weight := 0.0
	for _, v := range g.edges {
		if v.enabled == false {
			continue
		}
		fmt.Printf("Edge: %+v\n", v.value)
		count++
		weight += v.value.Weight()
	}

	return count, weight
}

func TestInvalidMST(t *testing.T) {
	graph := New()
	graph.AddVertex(node{"a"})
	graph.calculateMST()
	c, _ := printEnabledEdges(graph)
	if c != 0 {
		t.Fatalf("Unexpected MST: expected len=0, got=%v", c)
	}

	e := link{
		points: [2]point{point{"a", 1}, point{"b", 1}},
		weight: 2,
	}
	if err := graph.AddEdge(e); err == nil {
		t.Fatal("Expected error, but not occurred!")
	}
}

func TestRemoveVertex(t *testing.T) {
	graph := New()
	graph.AddVertex(node{"a"})
	graph.AddVertex(node{"b"})
	e := link{
		points: [2]point{point{"a", 1}, point{"b", 1}},
		weight: 2,
	}
	if err := graph.AddEdge(e); err != nil {
		t.Fatal(err)
	}
	graph.RemoveVertex(node{"a"})
	if len(graph.vertexies) != 1 {
		t.Fatalf("Expected node length is 1, got=%v\n", len(graph.vertexies))
	}
	if len(graph.edges) != 0 {
		t.Fatalf("Expected edge length is 0, got=%v\n", len(graph.edges))
	}
	if len(graph.points) != 0 {
		t.Fatalf("Expected points length is 0, got=%v\n", len(graph.points))
	}
	v := graph.vertexies["b"]
	if len(v.edges) != 0 {
		t.Fatalf("Expected edge length is 0, got=%v\n", len(v.edges))
	}
}

func TestRemoveEdges(t *testing.T) {
	graph := New()
	graph.AddVertex(node{"a"})
	graph.AddVertex(node{"b"})
	e := link{
		points: [2]point{point{"a", 1}, point{"b", 1}},
		weight: 2,
	}
	if err := graph.AddEdge(e); err != nil {
		t.Fatal(err)
	}
	graph.RemoveEdge(point{"a", 1})
	if len(graph.edges) != 0 {
		t.Fatalf("Expected edge length is 0, got=%v\n", len(graph.edges))
	}
	if len(graph.points) != 0 {
		t.Fatalf("Expected points length is 0, got=%v\n", len(graph.points))
	}
	a := graph.vertexies["a"]
	b := graph.vertexies["b"]
	if len(a.edges) != 0 || len(b.edges) != 0 {
		t.Fatalf("Expected # of edges is 0/0, got=%v/%v\n", len(a.edges), len(b.edges))
	}

	if err := graph.AddEdge(e); err != nil {
		t.Fatal(err)
	}
	if err := graph.AddEdge(e); err != nil {
		t.Fatal(err)
	}
	if len(a.edges) != 1 || len(b.edges) != 1 {
		t.Fatalf("Expected # of edges is 1/1, got=%v/%v\n", len(a.edges), len(b.edges))
	}
	graph.RemoveEdge(point{"a", 1})
	if len(graph.edges) != 0 {
		t.Fatalf("Expected edge length is 0, got=%v\n", len(graph.edges))
	}
	if len(graph.points) != 0 {
		t.Fatalf("Expected points length is 0, got=%v\n", len(graph.points))
	}
	if len(a.edges) != 0 || len(b.edges) != 0 {
		t.Fatalf("Expected # of edges is 0/0, got=%v/%v\n", len(a.edges), len(b.edges))
	}
}

func TestDuplicatedEdges(t *testing.T) {
	graph := New()
	graph.AddVertex(node{"a"})
	graph.AddVertex(node{"b"})
	e := link{
		points: [2]point{point{"a", 1}, point{"b", 1}},
		weight: 2,
	}
	if err := graph.AddEdge(e); err != nil {
		t.Fatal(err)
	}
	if err := graph.AddEdge(e); err != nil {
		t.Fatal(err)
	}

	graph.calculateMST()
	c, w := printEnabledEdges(graph)
	if c != 1 || w != 2 {
		t.Fatalf("Unexpected MST: expected=1/2, got=%v/%v", c, w)
	}
}

func TestMST0(t *testing.T) {
	graph := New()
	graph.AddVertex(node{"a"})
	graph.AddVertex(node{"b"})
	graph.AddVertex(node{"c"})
	graph.AddVertex(node{"d"})

	edges := make([]link, 0)
	edges = append(edges, link{
		points: [2]point{point{"a", 1}, point{"b", 1}},
		weight: 2,
	})
	edges = append(edges, link{
		points: [2]point{point{"c", 1}, point{"d", 3}},
		weight: 3,
	})

	for _, v := range edges {
		if err := graph.AddEdge(v); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 100; i++ {
		graph.calculateMST()
		c, w := printEnabledEdges(graph)
		if c != 2 || w != 5 {
			t.Fatalf("Unexpected MST: expected=2/5, got=%v/%v", c, w)
		}

		path := graph.FindPath(node{"a"}, node{"b"})
		fmt.Printf("Path: %+v\n", path)
		total := 0.0
		for _, v := range path {
			total += v.E.Weight()
		}
		if len(path) != 1 || total != 2 {
			t.Fatalf("Unexpected Path: expected=1/2, got=%v/%v", len(path), total)
		}

		path = graph.FindPath(node{"d"}, node{"c"})
		fmt.Printf("Path: %+v\n", path)
		total = 0.0
		for _, v := range path {
			total += v.E.Weight()
		}
		if len(path) != 1 || total != 3 {
			t.Fatalf("Unexpected Path: expected=1/3, got=%v/%v", len(path), total)
		}

		path = graph.FindPath(node{"b"}, node{"c"})
		fmt.Printf("Path: %+v\n", path)
		total = 0.0
		for _, v := range path {
			total += v.E.Weight()
		}
		if len(path) != 0 || total != 0 {
			t.Fatalf("Unexpected Path: expected=0/0, got=%v/%v", len(path), total)
		}
	}
}

func TestMST1(t *testing.T) {
	graph := New()
	graph.AddVertex(node{"a"})
	graph.AddVertex(node{"b"})
	graph.AddVertex(node{"c"})
	graph.AddVertex(node{"d"})

	edges := make([]link, 0)
	edges = append(edges, link{
		points: [2]point{point{"a", 1}, point{"b", 1}},
		weight: 2,
	})
	edges = append(edges, link{
		points: [2]point{point{"a", 2}, point{"d", 1}},
		weight: 1,
	})
	edges = append(edges, link{
		points: [2]point{point{"b", 2}, point{"d", 2}},
		weight: 3,
	})
	edges = append(edges, link{
		points: [2]point{point{"c", 1}, point{"d", 3}},
		weight: 3,
	})

	for _, v := range edges {
		if err := graph.AddEdge(v); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 100; i++ {
		graph.calculateMST()
		c, w := printEnabledEdges(graph)
		if c != 3 || w != 6 {
			t.Fatalf("Unexpected MST: expected=3/6, got=%v/%v", c, w)
		}

		path := graph.FindPath(node{"b"}, node{"c"})
		fmt.Printf("Path: %+v\n", path)
		total := 0.0
		for _, v := range path {
			total += v.E.Weight()
		}
		if len(path) != 3 || total != 6 {
			t.Fatalf("Unexpected Path: expected=3/6, got=%v/%v", len(path), total)
		}
	}
}

func TestMST2(t *testing.T) {
	graph := New()
	graph.AddVertex(node{"a"})
	graph.AddVertex(node{"b"})
	graph.AddVertex(node{"c"})
	graph.AddVertex(node{"d"})
	graph.AddVertex(node{"e"})
	graph.AddVertex(node{"f"})
	graph.AddVertex(node{"g"})

	edges := make([]link, 0)
	edges = append(edges, link{
		points: [2]point{point{"a", 1}, point{"b", 1}},
		weight: 4,
	})
	edges = append(edges, link{
		points: [2]point{point{"a", 2}, point{"c", 1}},
		weight: 8,
	})
	edges = append(edges, link{
		points: [2]point{point{"b", 2}, point{"c", 2}},
		weight: 9,
	})
	edges = append(edges, link{
		points: [2]point{point{"b", 1}, point{"d", 3}},
		weight: 8,
	})
	edges = append(edges, link{
		points: [2]point{point{"b", 1}, point{"e", 3}},
		weight: 10,
	})
	edges = append(edges, link{
		points: [2]point{point{"c", 1}, point{"d", 3}},
		weight: 2,
	})
	edges = append(edges, link{
		points: [2]point{point{"c", 1}, point{"f", 3}},
		weight: 1,
	})
	edges = append(edges, link{
		points: [2]point{point{"d", 1}, point{"e", 3}},
		weight: 7,
	})
	edges = append(edges, link{
		points: [2]point{point{"d", 1}, point{"f", 3}},
		weight: 9,
	})
	edges = append(edges, link{
		points: [2]point{point{"e", 1}, point{"f", 3}},
		weight: 5,
	})
	edges = append(edges, link{
		points: [2]point{point{"e", 1}, point{"g", 3}},
		weight: 6,
	})
	edges = append(edges, link{
		points: [2]point{point{"f", 1}, point{"g", 3}},
		weight: 2,
	})

	for _, v := range edges {
		if err := graph.AddEdge(v); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 100; i++ {
		graph.calculateMST()
		c, w := printEnabledEdges(graph)
		if c != 6 || w != 22 {
			t.Fatalf("Unexpected MST: expected=6/22, got=%v/%v", c, w)
		}

		path := graph.FindPath(node{"d"}, node{"e"})
		fmt.Printf("Path: %+v\n", path)
		total := 0.0
		for _, v := range path {
			total += v.E.Weight()
		}
		if len(path) != 3 || total != 8 {
			t.Fatalf("Unexpected Path: expected=3/8, got=%v/%v", len(path), total)
		}
	}
}

func TestMST3(t *testing.T) {
	graph := New()
	graph.AddVertex(node{"0"})
	graph.AddVertex(node{"1"})
	graph.AddVertex(node{"2"})
	graph.AddVertex(node{"3"})
	graph.AddVertex(node{"4"})
	graph.AddVertex(node{"5"})
	graph.AddVertex(node{"6"})
	graph.AddVertex(node{"7"})
	graph.AddVertex(node{"8"})

	edges := make([]link, 0)
	edges = append(edges, link{
		points: [2]point{point{"0", 1}, point{"1", 1}},
		weight: 4,
	})
	edges = append(edges, link{
		points: [2]point{point{"0", 2}, point{"7", 1}},
		weight: 8,
	})
	edges = append(edges, link{
		points: [2]point{point{"1", 2}, point{"2", 2}},
		weight: 9,
	})
	edges = append(edges, link{
		points: [2]point{point{"1", 1}, point{"7", 3}},
		weight: 11,
	})
	edges = append(edges, link{
		points: [2]point{point{"2", 1}, point{"3", 3}},
		weight: 7,
	})
	edges = append(edges, link{
		points: [2]point{point{"2", 1}, point{"5", 3}},
		weight: 4,
	})
	edges = append(edges, link{
		points: [2]point{point{"2", 1}, point{"8", 3}},
		weight: 2,
	})
	edges = append(edges, link{
		points: [2]point{point{"3", 1}, point{"4", 3}},
		weight: 9,
	})
	edges = append(edges, link{
		points: [2]point{point{"3", 1}, point{"5", 3}},
		weight: 14,
	})
	edges = append(edges, link{
		points: [2]point{point{"4", 1}, point{"5", 3}},
		weight: 10,
	})
	edges = append(edges, link{
		points: [2]point{point{"5", 1}, point{"6", 3}},
		weight: 2,
	})
	edges = append(edges, link{
		points: [2]point{point{"6", 1}, point{"7", 3}},
		weight: 1,
	})
	edges = append(edges, link{
		points: [2]point{point{"6", 1}, point{"8", 3}},
		weight: 6,
	})
	edges = append(edges, link{
		points: [2]point{point{"7", 1}, point{"8", 3}},
		weight: 7,
	})

	for _, v := range edges {
		if err := graph.AddEdge(v); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 100; i++ {
		graph.calculateMST()
		c, w := printEnabledEdges(graph)
		if c != 8 || w != 37 {
			t.Fatalf("Unexpected MST: expected=8/37, got=%v/%v", c, w)
		}

		path := graph.FindPath(node{"0"}, node{"8"})
		fmt.Printf("Path: %+v\n", path)
		total := 0.0
		for _, v := range path {
			total += v.E.Weight()
		}
		if len(path) != 5 || total != 17 {
			t.Fatalf("Unexpected Path: expected=5/17, got=%v/%v", len(path), total)
		}
	}
}

func TestMST4(t *testing.T) {
	graph := New()
	graph.AddVertex(node{"0"})
	graph.AddVertex(node{"1"})
	graph.AddVertex(node{"2"})
	graph.AddVertex(node{"3"})
	graph.AddVertex(node{"4"})
	graph.AddVertex(node{"5"})

	edges := make([]link, 0)
	edges = append(edges, link{
		points: [2]point{point{"0", 1}, point{"1", 1}},
		weight: 3,
	})
	edges = append(edges, link{
		points: [2]point{point{"0", 2}, point{"2", 1}},
		weight: 1,
	})
	edges = append(edges, link{
		points: [2]point{point{"0", 2}, point{"3", 2}},
		weight: 6,
	})
	edges = append(edges, link{
		points: [2]point{point{"1", 1}, point{"2", 3}},
		weight: 5,
	})
	edges = append(edges, link{
		points: [2]point{point{"1", 1}, point{"4", 3}},
		weight: 3,
	})
	edges = append(edges, link{
		points: [2]point{point{"2", 1}, point{"3", 3}},
		weight: 5,
	})
	edges = append(edges, link{
		points: [2]point{point{"2", 1}, point{"4", 3}},
		weight: 6,
	})
	edges = append(edges, link{
		points: [2]point{point{"2", 1}, point{"5", 3}},
		weight: 4,
	})
	edges = append(edges, link{
		points: [2]point{point{"3", 1}, point{"5", 3}},
		weight: 2,
	})
	edges = append(edges, link{
		points: [2]point{point{"4", 1}, point{"5", 3}},
		weight: 6,
	})

	for _, v := range edges {
		if err := graph.AddEdge(v); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 100; i++ {
		graph.calculateMST()
		c, w := printEnabledEdges(graph)
		if c != 5 || w != 13 {
			t.Fatalf("Unexpected MST: expected=5/13, got=%v/%v", c, w)
		}

		path := graph.FindPath(node{"1"}, node{"3"})
		fmt.Printf("Path: %+v\n", path)
		total := 0.0
		for _, v := range path {
			total += v.E.Weight()
		}
		if len(path) != 4 || total != 10 {
			t.Fatalf("Unexpected Path: expected=4/10, got=%v/%v", len(path), total)
		}
	}
}
