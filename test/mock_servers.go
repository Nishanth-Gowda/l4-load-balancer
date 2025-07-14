package test

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

// MockServer represents a simple HTTP server for testing
type MockServer struct {
	Port         int
	Name         string
	server       *http.Server
	RequestCount int64
}

// NewMockServer creates a new mock server
func NewMockServer(port int, name string) *MockServer {
	return &MockServer{
		Port: port,
		Name: name,
	}
}

// Start starts the mock server
func (ms *MockServer) Start() error {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "healthy", "server": "%s"}`, ms.Name)
	})

	// Main endpoint that tracks request counts
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&ms.RequestCount, 1)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"server": "%s", "port": %d, "request_count": %d, "timestamp": "%s"}`,
			ms.Name, ms.Port, count, time.Now().Format(time.RFC3339))
	})

	// Stats endpoint
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"server": "%s", "total_requests": %d}`,
			ms.Name, atomic.LoadInt64(&ms.RequestCount))
	})

	ms.server = &http.Server{
		Addr:    ":" + strconv.Itoa(ms.Port),
		Handler: mux,
	}

	log.Printf("Starting mock server %s on port %d", ms.Name, ms.Port)
	return ms.server.ListenAndServe()
}

// Stop stops the mock server
func (ms *MockServer) Stop() error {
	if ms.server != nil {
		return ms.server.Close()
	}
	return nil
}

// GetRequestCount returns the current request count
func (ms *MockServer) GetRequestCount() int64 {
	return atomic.LoadInt64(&ms.RequestCount)
}

// ResetRequestCount resets the request counter
func (ms *MockServer) ResetRequestCount() {
	atomic.StoreInt64(&ms.RequestCount, 0)
}

// MockServerPool manages multiple mock servers
type MockServerPool struct {
	servers []*MockServer
}

// NewMockServerPool creates a new pool of mock servers
func NewMockServerPool(ports []int) *MockServerPool {
	pool := &MockServerPool{
		servers: make([]*MockServer, len(ports)),
	}

	for i, port := range ports {
		pool.servers[i] = NewMockServer(port, fmt.Sprintf("server-%d", i+1))
	}

	return pool
}

// StartAll starts all mock servers in the pool
func (pool *MockServerPool) StartAll() {
	for _, server := range pool.servers {
		go func(s *MockServer) {
			if err := s.Start(); err != nil && err != http.ErrServerClosed {
				log.Printf("Mock server %s failed: %v", s.Name, err)
			}
		}(server)
	}

	// Wait a bit for servers to start
	time.Sleep(100 * time.Millisecond)
}

// StopAll stops all mock servers in the pool
func (pool *MockServerPool) StopAll() {
	for _, server := range pool.servers {
		if err := server.Stop(); err != nil {
			log.Printf("Error stopping server %s: %v", server.Name, err)
		}
	}
}

// GetServers returns all servers in the pool
func (pool *MockServerPool) GetServers() []*MockServer {
	return pool.servers
}

// GetTotalRequests returns the total request count across all servers
func (pool *MockServerPool) GetTotalRequests() int64 {
	var total int64
	for _, server := range pool.servers {
		total += server.GetRequestCount()
	}
	return total
}

// GetRequestDistribution returns a map of server names to request counts
func (pool *MockServerPool) GetRequestDistribution() map[string]int64 {
	distribution := make(map[string]int64)
	for _, server := range pool.servers {
		distribution[server.Name] = server.GetRequestCount()
	}
	return distribution
}

// ResetAllCounters resets request counters for all servers
func (pool *MockServerPool) ResetAllCounters() {
	for _, server := range pool.servers {
		server.ResetRequestCount()
	}
}

// WaitForHealthy waits for all servers to become healthy
func (pool *MockServerPool) WaitForHealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		allHealthy := true

		for _, server := range pool.servers {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", server.Port))
			if err != nil || resp.StatusCode != http.StatusOK {
				allHealthy = false
				if resp != nil {
					resp.Body.Close()
				}
				break
			}
			resp.Body.Close()
		}

		if allHealthy {
			return nil
		}

		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("servers did not become healthy within timeout")
}
