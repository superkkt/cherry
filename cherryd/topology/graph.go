/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package topology

import (
	"container/list"
	"errors"
	"sort"
	"sync"
)

type Vertex interface {
	ID() string
}

type Edge interface {
	Vertexies() [2]Vertex
	Weight() float64
	Compare(e Edge) bool
}

type node struct {
	vertex Vertex
	nEdges uint
}

type Graph struct {
	mutex sync.Mutex
	nodes map[string]*node
	edges *list.List
	mst   []Edge // Minimum Spanning Tree of this graph
}

func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[string]*node),
		edges: list.New(),
		mst:   make([]Edge, 0),
	}
}

func (r *Graph) AddVertex(v Vertex) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if v == nil {
		panic("Graph: adding nil vertex")
	}

	r.nodes[v.ID()] = &node{vertex: v}
}

func (r *Graph) RemoveVertex(v Vertex) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if v == nil {
		panic("Graph: removing nil vertex")
	}

	delete(r.nodes, v.ID())

	// Remove edges related with this vertex v
	var next *list.Element
	for elem := r.edges.Front(); elem != nil; elem = next {
		next = elem.Next()
		vertexies := elem.Value.(Edge).Vertexies()
		if vertexies[0].ID() != v.ID() && vertexies[1].ID() != v.ID() {
			continue
		}
		r.edges.Remove(elem)
	}
}

func (r *Graph) AddEdge(e Edge) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	vertexies := e.Vertexies()
	first, ok1 := r.nodes[vertexies[0].ID()]
	second, ok2 := r.nodes[vertexies[1].ID()]
	if !ok1 || !ok2 {
		return errors.New("Graph: adding an edge to unknown vertex")
	}
	first.nEdges++
	second.nEdges++

	r.edges.PushBack(e)

	return nil
}

func (r *Graph) RemoveEdge(e Edge) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	vertexies := e.Vertexies()
	first, ok1 := r.nodes[vertexies[0].ID()]
	second, ok2 := r.nodes[vertexies[1].ID()]
	if !ok1 || !ok2 {
		return errors.New("Graph: removing an edge to unknown vertex")
	}

	var next *list.Element
	for elem := r.edges.Front(); elem != nil; elem = next {
		next = elem.Next()
		v := elem.Value.(Edge)
		if !v.Compare(e) {
			continue
		}
		r.edges.Remove(elem)
		first.nEdges--
		second.nEdges--
	}

	return nil
}

type sortedEdge []Edge

func (r sortedEdge) Len() int {
	return len(r)
}

func (r sortedEdge) Less(i, j int) bool {
	if r[i].Weight() < r[j].Weight() {
		return true
	}

	return false
}

func (r sortedEdge) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r *Graph) pickRootVertex() Vertex {
	// Pick arbitrary vertex node
	for _, v := range r.nodes {
		if v.nEdges == 0 {
			continue
		}
		return v.vertex
	}

	return nil
}

func (r *Graph) makeSortedEdges() *list.List {
	edges := make(sortedEdge, r.edges.Len())
	for elem, i := r.edges.Front(), 0; elem != nil; elem, i = elem.Next(), i+1 {
		edges[i] = elem.Value.(Edge)
	}
	sort.Sort(edges)

	result := list.New()
	for _, v := range edges {
		result.PushBack(v)
	}

	return result
}

func (r *Graph) pickValidVertexies() []Vertex {
	result := make([]Vertex, 0)
	for _, v := range r.nodes {
		if v.nEdges == 0 {
			continue
		}
		result = append(result, v.vertex)
	}

	return result
}

// calculateMST finds a minimum spanning tree of this graph. A caller should lock the mutex before calling this function.
func (r *Graph) CalculateMST() {
	if r.edges.Len() == 0 || len(r.nodes) == 0 {
		return
	}

	edges := r.makeSortedEdges()
	Vp := make(map[string]Vertex)
	Ep := make([]Edge, 0)

	root := r.pickRootVertex()
	if root == nil {
		// There is no spanning tree for this graph.
		r.mst = make([]Edge, 0)
		return
	}
	// Initial vertex node
	Vp[root.ID()] = root
	V := r.pickValidVertexies()

	for len(V) != len(Vp) {
		var next *list.Element
		for elem := edges.Front(); elem != nil; elem = next {
			next = elem.Next()
			e := elem.Value.(Edge)
			vertexies := e.Vertexies()
			_, left := Vp[vertexies[0].ID()]
			_, right := Vp[vertexies[1].ID()]
			if (left && right) || (!left && !right) {
				continue
			}
			if left {
				Vp[vertexies[1].ID()] = vertexies[1]
			} else {
				Vp[vertexies[0].ID()] = vertexies[0]
			}
			Ep = append(Ep, e)
			edges.Remove(elem)
			break
		}
	}
	r.mst = Ep
}
