/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package topology

import (
	"fmt"
	"testing"
)

type vertex struct {
	dpid string
}

func (r vertex) ID() string {
	return fmt.Sprintf("%v", r.dpid)
}

type link struct {
	dpid string
	port uint32
}

type edge struct {
	links  [2]link
	weight float64
}

func (r edge) Vertexies() [2]Vertex {
	v1 := vertex{r.links[0].dpid}
	v2 := vertex{r.links[1].dpid}
	var result [2]Vertex
	result[0] = v1
	result[1] = v2
	return result
}
func (r edge) Weight() float64 {
	return r.weight
}

func (r edge) Compare(e Edge) bool {
	t := e.(edge)
	c0 := r.links[0].dpid == t.links[0].dpid
	c1 := r.links[1].dpid == t.links[1].dpid
	c2 := r.links[0].port == t.links[0].port
	c3 := r.links[1].port == t.links[1].port

	if c0 && c1 && c2 && c3 {
		return true
	}

	return false
}

func TestInvalidMST(t *testing.T) {
	graph := NewGraph()
	graph.CalculateMST()
	fmt.Printf("MST: %+v\n", graph.mst)
	if len(graph.mst) != 0 {
		t.Fatalf("Unexpected MST: expected len=0, got=%v", len(graph.mst))
	}

	graph.AddVertex(vertex{"a"})
	graph.CalculateMST()
	fmt.Printf("MST: %+v\n", graph.mst)
	if len(graph.mst) != 0 {
		t.Fatalf("Unexpected MST: expected len=0, got=%v", len(graph.mst))
	}

	e := edge{
		links:  [2]link{link{"a", 1}, link{"b", 1}},
		weight: 2,
	}
	if err := graph.AddEdge(e); err == nil {
		t.Fatal("Expected error, but not occurred!")
	}
}

func TestRemoveVertex(t *testing.T) {
	graph := NewGraph()
	graph.AddVertex(vertex{"a"})
	graph.AddVertex(vertex{"b"})
	e := edge{
		links:  [2]link{link{"a", 1}, link{"b", 1}},
		weight: 2,
	}
	if err := graph.AddEdge(e); err != nil {
		t.Fatal(err)
	}
	graph.RemoveVertex(vertex{"a"})
	if graph.edges.Len() != 0 {
		t.Fatalf("Expected node length is 0, got=%v\n", graph.edges.Len())
	}
}

func TestRemoveEdges(t *testing.T) {
	graph := NewGraph()
	graph.AddVertex(vertex{"a"})
	graph.AddVertex(vertex{"b"})
	e := edge{
		links:  [2]link{link{"a", 1}, link{"b", 1}},
		weight: 2,
	}
	if err := graph.AddEdge(e); err != nil {
		t.Fatal(err)
	}
	graph.RemoveEdge(e)
	if graph.edges.Len() != 0 {
		t.Fatalf("Expected node length is 0, got=%v\n", graph.edges.Len())
	}
	a := graph.nodes["a"]
	b := graph.nodes["b"]
	if a.nEdges != 0 || b.nEdges != 0 {
		t.Fatalf("Expected # of edges is 0/0, got=%v/%v\n", a.nEdges, b.nEdges)
	}

	if err := graph.AddEdge(e); err != nil {
		t.Fatal(err)
	}
	if err := graph.AddEdge(e); err != nil {
		t.Fatal(err)
	}
	if a.nEdges != 2 || b.nEdges != 2 {
		t.Fatalf("Expected # of edges is 2/2, got=%v/%v\n", a.nEdges, b.nEdges)
	}
	graph.RemoveEdge(e)
	if graph.edges.Len() != 0 {
		t.Fatalf("Expected node length is 0, got=%v\n", graph.edges.Len())
	}
	if a.nEdges != 0 || b.nEdges != 0 {
		t.Fatalf("Expected # of edges is 0/0, got=%v/%v\n", a.nEdges, b.nEdges)
	}
}

func TestDuplicatedEdges(t *testing.T) {
	graph := NewGraph()
	graph.AddVertex(vertex{"a"})
	graph.AddVertex(vertex{"b"})
	e := edge{
		links:  [2]link{link{"a", 1}, link{"b", 1}},
		weight: 2,
	}
	if err := graph.AddEdge(e); err != nil {
		t.Fatal(err)
	}
	if err := graph.AddEdge(e); err != nil {
		t.Fatal(err)
	}

	graph.CalculateMST()
	fmt.Printf("MST: %+v\n", graph.mst)
	total := 0.0
	for _, v := range graph.mst {
		total += v.Weight()
	}
	if len(graph.mst) != 1 || total != 2 {
		t.Fatalf("Unexpected MST: expected=1/2, got=%v/%v", len(graph.mst), total)
	}
}

func TestMST1(t *testing.T) {
	graph := NewGraph()
	graph.AddVertex(vertex{"a"})
	graph.AddVertex(vertex{"b"})
	graph.AddVertex(vertex{"c"})
	graph.AddVertex(vertex{"d"})

	edges := make([]edge, 0)
	edges = append(edges, edge{
		links:  [2]link{link{"a", 1}, link{"b", 1}},
		weight: 2,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"a", 2}, link{"d", 1}},
		weight: 1,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"b", 2}, link{"d", 2}},
		weight: 2,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"c", 1}, link{"d", 3}},
		weight: 3,
	})

	for _, v := range edges {
		if err := graph.AddEdge(v); err != nil {
			t.Fatal(err)
		}
	}

	graph.CalculateMST()
	fmt.Printf("MST: %+v\n", graph.mst)
	total := 0.0
	for _, v := range graph.mst {
		total += v.Weight()
	}
	if len(graph.mst) != 3 || total != 6 {
		t.Fatalf("Unexpected MST: expected=3/6, got=%v/%v", len(graph.mst), total)
	}
}

func TestMST2(t *testing.T) {
	graph := NewGraph()
	graph.AddVertex(vertex{"a"})
	graph.AddVertex(vertex{"b"})
	graph.AddVertex(vertex{"c"})
	graph.AddVertex(vertex{"d"})
	graph.AddVertex(vertex{"e"})
	graph.AddVertex(vertex{"f"})
	graph.AddVertex(vertex{"g"})

	edges := make([]edge, 0)
	edges = append(edges, edge{
		links:  [2]link{link{"a", 1}, link{"b", 1}},
		weight: 4,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"a", 2}, link{"c", 1}},
		weight: 8,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"b", 2}, link{"c", 2}},
		weight: 9,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"b", 1}, link{"d", 3}},
		weight: 8,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"b", 1}, link{"e", 3}},
		weight: 10,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"c", 1}, link{"d", 3}},
		weight: 2,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"c", 1}, link{"f", 3}},
		weight: 1,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"d", 1}, link{"e", 3}},
		weight: 7,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"d", 1}, link{"f", 3}},
		weight: 9,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"e", 1}, link{"f", 3}},
		weight: 5,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"e", 1}, link{"g", 3}},
		weight: 6,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"f", 1}, link{"g", 3}},
		weight: 2,
	})

	for _, v := range edges {
		if err := graph.AddEdge(v); err != nil {
			t.Fatal(err)
		}
	}

	graph.CalculateMST()
	fmt.Printf("MST: %+v\n", graph.mst)
	total := 0.0
	for _, v := range graph.mst {
		total += v.Weight()
	}
	if len(graph.mst) != 6 || total != 22 {
		t.Fatalf("Unexpected MST: expected=6/22, got=%v/%v", len(graph.mst), total)
	}
}

func TestMST3(t *testing.T) {
	graph := NewGraph()
	graph.AddVertex(vertex{"0"})
	graph.AddVertex(vertex{"1"})
	graph.AddVertex(vertex{"2"})
	graph.AddVertex(vertex{"3"})
	graph.AddVertex(vertex{"4"})
	graph.AddVertex(vertex{"5"})
	graph.AddVertex(vertex{"6"})
	graph.AddVertex(vertex{"7"})
	graph.AddVertex(vertex{"8"})

	edges := make([]edge, 0)
	edges = append(edges, edge{
		links:  [2]link{link{"0", 1}, link{"1", 1}},
		weight: 4,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"0", 2}, link{"7", 1}},
		weight: 8,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"1", 2}, link{"2", 2}},
		weight: 8,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"1", 1}, link{"7", 3}},
		weight: 11,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"2", 1}, link{"3", 3}},
		weight: 7,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"2", 1}, link{"5", 3}},
		weight: 4,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"2", 1}, link{"8", 3}},
		weight: 2,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"3", 1}, link{"4", 3}},
		weight: 9,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"3", 1}, link{"5", 3}},
		weight: 14,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"4", 1}, link{"5", 3}},
		weight: 10,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"5", 1}, link{"6", 3}},
		weight: 2,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"6", 1}, link{"7", 3}},
		weight: 1,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"6", 1}, link{"8", 3}},
		weight: 6,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"7", 1}, link{"8", 3}},
		weight: 7,
	})

	for _, v := range edges {
		if err := graph.AddEdge(v); err != nil {
			t.Fatal(err)
		}
	}

	graph.CalculateMST()
	fmt.Printf("MST: %+v\n", graph.mst)
	total := 0.0
	for _, v := range graph.mst {
		total += v.Weight()
	}
	if len(graph.mst) != 8 || total != 37 {
		t.Fatalf("Unexpected MST: expected=8/37, got=%v/%v", len(graph.mst), total)
	}
}

func TestMST4(t *testing.T) {
	graph := NewGraph()
	graph.AddVertex(vertex{"0"})
	graph.AddVertex(vertex{"1"})
	graph.AddVertex(vertex{"2"})
	graph.AddVertex(vertex{"3"})
	graph.AddVertex(vertex{"4"})
	graph.AddVertex(vertex{"5"})

	edges := make([]edge, 0)
	edges = append(edges, edge{
		links:  [2]link{link{"0", 1}, link{"1", 1}},
		weight: 3,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"0", 2}, link{"2", 1}},
		weight: 1,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"0", 2}, link{"3", 2}},
		weight: 6,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"1", 1}, link{"2", 3}},
		weight: 5,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"1", 1}, link{"4", 3}},
		weight: 3,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"2", 1}, link{"3", 3}},
		weight: 5,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"2", 1}, link{"4", 3}},
		weight: 6,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"2", 1}, link{"5", 3}},
		weight: 4,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"3", 1}, link{"5", 3}},
		weight: 2,
	})
	edges = append(edges, edge{
		links:  [2]link{link{"4", 1}, link{"5", 3}},
		weight: 6,
	})

	for _, v := range edges {
		if err := graph.AddEdge(v); err != nil {
			t.Fatal(err)
		}
	}

	graph.CalculateMST()
	fmt.Printf("MST: %+v\n", graph.mst)
	total := 0.0
	for _, v := range graph.mst {
		total += v.Weight()
	}
	if len(graph.mst) != 5 || total != 13 {
		t.Fatalf("Unexpected MST: expected=5/13, got=%v/%v", len(graph.mst), total)
	}
}
