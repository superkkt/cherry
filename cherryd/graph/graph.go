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

// Point is a spot on a vertex. We need this to represent multiple links among two vertexies.
type Point interface {
	ID() string
	Vertex() Vertex
}

// Edge is a link, which has a weight, among two points.
type Edge interface {
	ID() string
	Points() [2]Point
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
	mutex     sync.RWMutex
	vertexies map[string]vertex
	edges     map[string]*edge
	points    map[string]*edge
}

func New() *Graph {
	return &Graph{
		vertexies: make(map[string]vertex),
		edges:     make(map[string]*edge),
		points:    make(map[string]*edge),
	}
}

func (r *Graph) AddVertex(v Vertex) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if v == nil {
		panic("adding nil vertex")
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
	v := e.Points()
	v1, ok := r.vertexies[v[0].Vertex().ID()]
	if ok {
		delete(v1.edges, e.ID())
	}
	v2, ok := r.vertexies[v[1].Vertex().ID()]
	if ok {
		delete(v2.edges, e.ID())
	}
	delete(r.edges, e.ID())
	delete(r.points, v[0].ID())
	delete(r.points, v[1].ID())
}

func (r *Graph) RemoveVertex(v Vertex) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if v == nil {
		panic("removing nil vertex")
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
		panic("adding nil edge")
	}
	// Check duplication
	_, ok := r.edges[e.ID()]
	if ok {
		return nil
	}

	points := e.Points()
	if points[0].Vertex() == nil || points[1].Vertex() == nil {
		panic("adding an edge pointing to nil vertex")
	}
	first, ok1 := r.vertexies[points[0].Vertex().ID()]
	second, ok2 := r.vertexies[points[1].Vertex().ID()]
	if !ok1 || !ok2 {
		return errors.New("AddEdge: adding an edge to unknown vertex")
	}

	edge := &edge{value: e}
	r.edges[e.ID()] = edge
	first.edges[e.ID()] = edge
	second.edges[e.ID()] = edge
	r.points[points[0].ID()] = edge
	r.points[points[1].ID()] = edge
	r.calculateMST()

	return nil
}

func (r *Graph) RemoveEdge(p Point) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	e, ok := r.points[p.ID()]
	if !ok {
		return
	}
	r.removeEdge(e.value)
	r.calculateMST()
}

// IsEdge returns whether p is on an edge between two vertexeis.
func (r *Graph) IsEdge(p Point) bool {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if p == nil {
		panic("nil point")
	}

	_, ok := r.points[p.ID()]
	return ok
}

// IsEnabledPoint returns whether p is an active point that is not disabled by the minimum spanning tree.
func (r *Graph) IsEnabledPoint(p Point) bool {
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if p == nil {
		panic("nil point")
	}

	v, ok := r.points[p.ID()]
	if !ok {
		return false
	}

	return v.enabled
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

		points := e.value.Points()
		v1, ok := clusters[points[0].Vertex().ID()]
		if !ok {
			panic("invalid edge pointing an unknown vertex")
		}
		v2, ok := clusters[points[1].Vertex().ID()]
		if !ok {
			panic("invalid edge pointing an unknown vertex")
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
	// Read lock
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if len(r.vertexies) == 0 || len(r.edges) == 0 {
		return []Path{}
	}

	visited := make(map[string]bool)
	prev := make(map[string]Path)

	queue := newQueue()
	queue.enqueue(src)
	visited[src.ID()] = true

	// Implementation of BFS algorithm
	for queue.length() > 0 {
		v := queue.dequeue()
		if v == nil {
			panic("nil element is fetched from the queue")
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
			points := w.value.Points()
			next := points[0]
			if points[0].Vertex().ID() == vertex.value.ID() {
				next = points[1]
			}
			if _, ok := visited[next.Vertex().ID()]; ok {
				continue
			}
			visited[next.Vertex().ID()] = true
			prev[next.Vertex().ID()] = Path{V: vertex.value, E: w.value}
			queue.enqueue(next.Vertex())
		}
	}

	u := dst
	result := make([]Path, 0)
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
