package balancer

import (
	"net"
)

// LoadBalancer represents the main load balancer
type LoadBalancer struct {
	listenAddr string
	backends   []Backend
	algorithm  Algorithm
}

// Backend represents a backend server
type Backend struct {
	Address string
	Healthy bool
}

// Algorithm interface for load balancing algorithms
type Algorithm interface {
	SelectBackend(backends []Backend) *Backend
}

// NewLoadBalancer creates a new load balancer instance
func NewLoadBalancer(listenAddr string, backends []Backend, algorithm Algorithm) *LoadBalancer {
	return &LoadBalancer{
		listenAddr: listenAddr,
		backends:   backends,
		algorithm:  algorithm,
	}
}

// Start starts the load balancer server
func (lb *LoadBalancer) Start() error {
	// TODO: Implement L4 load balancing logic
	listener, err := net.Listen("tcp", lb.listenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go lb.handleConnection(conn)
	}
}

// handleConnection handles incoming connections
func (lb *LoadBalancer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// TODO: Select backend using algorithm
	// TODO: Proxy connection to selected backend
}
