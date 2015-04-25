/*
 * Cherry - An OpenFlow Controller
 *
 * Copyright (C) 2015 Samjung Data Service Co., Ltd.,
 * Kitae Kim <superkkt@sds.co.kr>
 */

package graph

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
	ID() string
	Vertexies() [2]Vertex
	Weight() float64
}

type edge struct {
	value   Edge
	enabled bool
}

type vertex struct {
	value Vertex
	edges map[string]*edge
}

type Graph struct {
	mutex     sync.Mutex
	vertexies map[string]vertex
	edges     map[string]*edge
}

func New() *Graph {
	return &Graph{
		vertexies: make(map[string]vertex),
		edges:     make(map[string]*edge),
	}
}

func (r *Graph) AddVertex(v Vertex) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if v == nil {
		panic("Graph: adding nil vertex")
	}
	// Check duplication
	_, ok := r.vertexies[v.ID()]
	if ok {
		return
	}

	r.vertexies[v.ID()] = vertex{
		value: v,
		edges: make(map[string]*edge),
	}
	r.calculateMST()
}

func (r *Graph) removeEdge(e Edge) {
	v := e.Vertexies()
	v1, ok := r.vertexies[v[0].ID()]
	if ok {
		delete(v1.edges, e.ID())
	}
	v2, ok := r.vertexies[v[1].ID()]
	if ok {
		delete(v2.edges, e.ID())
	}
	delete(r.edges, e.ID())
}

func (r *Graph) RemoveVertex(v Vertex) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if v == nil {
		panic("Graph: removing nil vertex")
	}

	vertex, ok := r.vertexies[v.ID()]
	if !ok {
		return
	}
	for _, e := range vertex.edges {
		r.removeEdge(e.value)
	}
	delete(r.vertexies, v.ID())
	r.calculateMST()
}

func (r *Graph) AddEdge(e Edge) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if e == nil {
		panic("Graph: adding nil edge")
	}
	// Check duplication
	_, ok := r.edges[e.ID()]
	if ok {
		return nil
	}

	vertexies := e.Vertexies()
	if vertexies[0] == nil || vertexies[1] == nil {
		panic("Graph: adding an edge pointing to nil vertex")
	}
	first, ok1 := r.vertexies[vertexies[0].ID()]
	second, ok2 := r.vertexies[vertexies[1].ID()]
	if !ok1 || !ok2 {
		return errors.New("Graph: adding an edge to unknown vertex")
	}

	edge := &edge{value: e}
	r.edges[e.ID()] = edge
	first.edges[e.ID()] = edge
	second.edges[e.ID()] = edge
	r.calculateMST()

	return nil
}

func (r *Graph) RemoveEdge(e Edge) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if e == nil {
		panic("Graph: removing nil edge")
	}
	_, ok := r.edges[e.ID()]
	if !ok {
		return nil
	}

	vertexies := e.Vertexies()
	if vertexies[0] == nil || vertexies[1] == nil {
		panic("Graph: removing an edge pointing to nil vertex")
	}
	first, ok1 := r.vertexies[vertexies[0].ID()]
	second, ok2 := r.vertexies[vertexies[1].ID()]
	if !ok1 || !ok2 {
		return errors.New("Graph: removing an edge to unknown vertex")
	}

	delete(first.edges, e.ID())
	delete(second.edges, e.ID())
	delete(r.edges, e.ID())
	r.calculateMST()

	return nil
}

type sortedEdge []*edge

func (r sortedEdge) Len() int {
	return len(r)
}

func (r sortedEdge) Less(i, j int) bool {
	if r[i].value.Weight() < r[j].value.Weight() {
		return true
	}

	return false
}

func (r sortedEdge) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r *Graph) pickRootVertex() Vertex {
	// Pick arbitrary vertex node that has at least one edge.
	for _, v := range r.vertexies {
		if len(v.edges) == 0 {
			continue
		}
		return v.value
	}

	return nil
}

func (r *Graph) resetEdges() *list.List {
	edges := make(sortedEdge, 0)
	for _, v := range r.edges {
		// Disable all edges
		v.enabled = false
		edges = append(edges, v)
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
	for _, v := range r.vertexies {
		if len(v.edges) == 0 {
			continue
		}
		result = append(result, v.value)
	}

	return result
}

func (r *Graph) makeClusters() map[string]*list.List {
	result := make(map[string]*list.List)
	for _, v := range r.vertexies {
		l := list.New()
		l.PushBack(v)
		result[v.value.ID()] = l
	}

	return result
}

func mergeCluster(clusters map[string]*list.List, l1, l2 *list.List) {
	v := list.New()
	v.PushBackList(l1)
	v.PushBackList(l2)

	for elem := v.Front(); elem != nil; elem = elem.Next() {
		vertex := elem.Value.(vertex)
		clusters[vertex.value.ID()] = v
	}
}

// calculateMST finds a minimum spanning tree of this graph using Kruskal's algorithm.
// A caller should lock the mutex before calling this function.
func (r *Graph) calculateMST() {
	if len(r.edges) == 0 || len(r.vertexies) == 0 {
		return
	}

	// FIXME: Use priority queue instead of sorting!
	edges := r.resetEdges()
	clusters := r.makeClusters()

	count := 0
	for count < len(r.vertexies)-1 {
		if edges.Len() == 0 {
			break
		}

		// Pop the minimum weighted edge
		elem := edges.Front()
		e := elem.Value.(*edge)
		edges.Remove(elem)

		vertexies := e.value.Vertexies()
		v1, ok := clusters[vertexies[0].ID()]
		if !ok {
			panic("Graph: invalid edge pointing an unknown vertex")
		}
		v2, ok := clusters[vertexies[1].ID()]
		if !ok {
			panic("Graph: invalid edge pointing an unknown vertex")
		}

		// Prevent a loop
		if v1 == v2 {
			continue
		}
		// Found new edge to be included in MST
		mergeCluster(clusters, v1, v2)
		e.enabled = true
		count++
	}
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

	if len(r.vertexies) == 0 || len(r.edges) == 0 {
		return []Path{}
	}

	result := make([]Path, 0)
	visited := make(map[string]bool)
	prev := make(map[string]Path)

	queue := newQueue()
	queue.enqueue(src)
	visited[src.ID()] = true

	// Implementation of BFS algorithm
	for queue.length() > 0 {
		v := queue.dequeue()
		if v == nil {
			panic("Graph: nil element is fetched from the queue")
		}

		vertex, ok := r.vertexies[v.(Vertex).ID()]
		if !ok {
			return []Path{}
		}
		for _, w := range vertex.edges {
			// We only use edges that belong to MST.
			if w.enabled == false {
				continue
			}
			vertexies := w.value.Vertexies()
			next := vertexies[0]
			if vertexies[0].ID() == vertex.value.ID() {
				next = vertexies[1]
			}
			if _, ok := visited[next.ID()]; ok {
				continue
			}
			visited[next.ID()] = true
			prev[next.ID()] = Path{V: vertex.value, E: w.value}
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
