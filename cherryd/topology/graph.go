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
	edges  map[*list.Element]Edge
	mst    []Edge // Edges that belongs to the MST. This is updated when CalculateMST() is called.
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
	r.nodes[v.ID()] = &node{
		vertex: v,
		edges:  make(map[*list.Element]Edge),
	}
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

	elem := r.edges.PushBack(e)
	first.edges[elem] = e
	second.edges[elem] = e

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
		delete(first.edges, elem)
		delete(second.edges, elem)
		r.edges.Remove(elem)
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
		if len(v.edges) == 0 {
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
		if len(v.edges) == 0 {
			continue
		}
		result = append(result, v.vertex)
	}

	return result
}

func (r *Graph) resetNodeEdges() {
	for _, v := range r.nodes {
		v.mst = make([]Edge, 0)
	}
}

// calculateMST finds a minimum spanning tree of this graph. A caller should lock the mutex before calling this function.
func (r *Graph) CalculateMST() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.edges.Len() == 0 || len(r.nodes) == 0 {
		return
	}

	edges := r.makeSortedEdges()
	Vp := make(map[string]Vertex)
	Ep := make([]Edge, 0)

	r.resetNodeEdges()
	V := r.pickValidVertexies()
	root := r.pickRootVertex()
	if root == nil {
		// There is no spanning tree for this graph.
		goto finish
	}
	// Initial vertex node
	Vp[root.ID()] = root

	for len(V) != len(Vp) {
		var next *list.Element
		for elem := edges.Front(); elem != nil; elem = next {
			next = elem.Next()
			e := elem.Value.(Edge)
			vertexies := e.Vertexies()
			_, first := Vp[vertexies[0].ID()]
			_, second := Vp[vertexies[1].ID()]
			if (first && second) || (!first && !second) {
				continue
			}
			if first {
				Vp[vertexies[1].ID()] = vertexies[1]
			} else {
				Vp[vertexies[0].ID()] = vertexies[0]
			}
			r.updateMSTEdge(vertexies, e)
			Ep = append(Ep, e)
			edges.Remove(elem)
			break
		}
	}

finish:
	r.mst = Ep
}

func (r *Graph) updateMSTEdge(vertexies [2]Vertex, e Edge) {
	first, ok := r.nodes[vertexies[0].ID()]
	if !ok {
		panic("Graph: trying to update MST edge for an unknown node")
	}
	second, ok := r.nodes[vertexies[1].ID()]
	if !ok {
		panic("Graph: trying to update MST edge for an unknown node")
	}

	first.mst = append(first.mst, e)
	second.mst = append(second.mst, e)
}

type queue struct {
	list *list.List
}

func newQueue() *queue {
	return &queue{list.New()}
}

func (r *queue) enqueue(v interface{}) {
	r.list.PushBack(v)
}

func (r *queue) dequeue() interface{} {
	v := r.list.Front()
	if v != nil {
		r.list.Remove(v)
	}
	return v.Value
}

func (r *queue) length() int {
	return r.list.Len()
}

type Path struct {
	V Vertex
	E Edge
}

func (r *Graph) FindPath(src, dst Vertex) []Path {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	result := make([]Path, 0)
	// # of vertexies in MST should be greater than 1.
	if len(r.mst) <= 1 {
		return result
	}

	visited := make(map[string]bool)
	prev := make(map[string]Path)

	queue := newQueue()
	queue.enqueue(src)
	visited[src.ID()] = true

	for queue.length() > 0 {
		v := queue.dequeue()
		if v == nil {
			panic("Graph: nil element is fetched from the queue")
		}

		node, ok := r.nodes[v.(Vertex).ID()]
		if !ok {
			panic("Graph: unknown vertex node")
		}
		for _, w := range node.mst {
			vertexies := w.Vertexies()
			next := vertexies[0]
			if vertexies[0].ID() == node.vertex.ID() {
				next = vertexies[1]
			}
			if _, ok := visited[next.ID()]; ok {
				continue
			}
			visited[next.ID()] = true
			prev[next.ID()] = Path{V: node.vertex, E: w}
			queue.enqueue(next)
		}
	}

	u := dst
	for {
		path, ok := prev[u.ID()]
		if !ok {
			break
		}
		result = append(result, path)
		u = path.V
	}

	return reverse(result)
}

func reverse(data []Path) []Path {
	length := len(data)
	if length == 0 {
		return data
	}

	result := make([]Path, length)
	for i, j := 0, length-1; i < length; i, j = i+1, j-1 {
		result[i] = data[j]
	}

	return result
}
