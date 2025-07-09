package balancer

import (
	"sync/atomic"
)

// RoundRobinAlgorithm implements round-robin load balancing
type RoundRobinAlgorithm struct {
	counter uint64
}

// NewRoundRobinAlgorithm creates a new round-robin algorithm
func NewRoundRobinAlgorithm() *RoundRobinAlgorithm {
	return &RoundRobinAlgorithm{}
}

// SelectBackend selects the next backend using round-robin
func (rr *RoundRobinAlgorithm) SelectBackend(backends []Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}

	// Filter healthy backends
	healthy := make([]Backend, 0, len(backends))
	for _, backend := range backends {
		if backend.Healthy {
			healthy = append(healthy, backend)
		}
	}

	if len(healthy) == 0 {
		return nil
	}

	index := atomic.AddUint64(&rr.counter, 1) % uint64(len(healthy))
	return &healthy[index]
}

// LeastConnectionsAlgorithm implements least connections load balancing
type LeastConnectionsAlgorithm struct {
	// TODO: Track connections per backend
}

// NewLeastConnectionsAlgorithm creates a new least connections algorithm
func NewLeastConnectionsAlgorithm() *LeastConnectionsAlgorithm {
	return &LeastConnectionsAlgorithm{}
}

// SelectBackend selects the backend with least connections
func (lc *LeastConnectionsAlgorithm) SelectBackend(backends []Backend) *Backend {
	// TODO: Implement least connections logic
	return nil
}
