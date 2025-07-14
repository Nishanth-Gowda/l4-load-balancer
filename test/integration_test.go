package test

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"l4-load-balancer/internal/backend"
	"l4-load-balancer/internal/balancer"
	"l4-load-balancer/internal/health"
)

func TestLoadBalancerIntegration(t *testing.T) {
	// Start mock backend servers
	backendPorts := []int{8081, 8082, 8083}
	pool := NewMockServerPool(backendPorts)
	pool.StartAll()
	defer pool.StopAll()

	// Wait for servers to be ready
	if err := pool.WaitForHealthy(5 * time.Second); err != nil {
		t.Fatalf("Mock servers failed to start: %v", err)
	}

	// Create backend manager
	manager := backend.NewManager()
	for _, port := range backendPorts {
		manager.AddServer("localhost", port)
	}

	// Create health checker
	checker := health.NewChecker(manager, 1*time.Second, 1*time.Second)
	go checker.Start()
	defer checker.Stop()

	// Wait for health checks to complete
	time.Sleep(2 * time.Second)

	// Create load balancer with round-robin algorithm
	algorithm := balancer.NewRoundRobinAlgorithm()

	// Convert backend servers to balancer backends
	backends := make([]balancer.Backend, 0)
	for _, server := range manager.GetHealthyServers() {
		backends = append(backends, balancer.Backend{
			Address: server.GetAddress(),
			Healthy: server.Healthy,
		})
	}

	if len(backends) != len(backendPorts) {
		t.Fatalf("Expected %d healthy backends, got %d", len(backendPorts), len(backends))
	}

	// Test round-robin distribution
	requestCount := 30
	backendCounts := make(map[string]int)

	for i := 0; i < requestCount; i++ {
		backend := algorithm.SelectBackend(backends)
		if backend == nil {
			t.Fatalf("Algorithm returned nil backend for request %d", i)
		}
		backendCounts[backend.Address]++
	}

	// Verify roughly even distribution (should be 10 requests per backend)
	expectedPerBackend := requestCount / len(backends)
	tolerance := 2 // Allow some variance due to round-robin timing

	for _, backend := range backends {
		count := backendCounts[backend.Address]
		if count < expectedPerBackend-tolerance || count > expectedPerBackend+tolerance {
			t.Errorf("Backend %s: expected ~%d requests, got %d",
				backend.Address, expectedPerBackend, count)
		}
	}

	// Verify all backends received requests
	if len(backendCounts) != len(backends) {
		t.Errorf("Expected %d backends to receive requests, but %d did",
			len(backends), len(backendCounts))
	}
}

func TestLoadBalancerWithFailingBackend(t *testing.T) {
	// Start mock backend servers
	backendPorts := []int{8084, 8085, 8086}
	pool := NewMockServerPool(backendPorts)
	pool.StartAll()
	defer pool.StopAll()

	// Wait for servers to be ready
	if err := pool.WaitForHealthy(5 * time.Second); err != nil {
		t.Fatalf("Mock servers failed to start: %v", err)
	}

	// Create backend manager
	manager := backend.NewManager()
	for _, port := range backendPorts {
		manager.AddServer("localhost", port)
	}

	// Create health checker
	checker := health.NewChecker(manager, 500*time.Millisecond, 1*time.Second)
	go checker.Start()
	defer checker.Stop()

	// Wait for initial health checks
	time.Sleep(1 * time.Second)

	// Verify all servers are initially healthy
	healthyServers := manager.GetHealthyServers()
	if len(healthyServers) != len(backendPorts) {
		t.Fatalf("Expected %d healthy servers, got %d", len(backendPorts), len(healthyServers))
	}

	// Stop one backend server to simulate failure
	servers := pool.GetServers()
	if err := servers[1].Stop(); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	// Wait for health checker to detect the failure
	time.Sleep(2 * time.Second)

	// Verify that only 2 servers are now healthy
	healthyServers = manager.GetHealthyServers()
	if len(healthyServers) != 2 {
		t.Errorf("Expected 2 healthy servers after failure, got %d", len(healthyServers))
	}

	// Test that load balancer only selects healthy backends
	algorithm := balancer.NewRoundRobinAlgorithm()
	backends := make([]balancer.Backend, 0)
	for _, server := range manager.GetAllServers() {
		backends = append(backends, balancer.Backend{
			Address: server.GetAddress(),
			Healthy: server.Healthy,
		})
	}

	// Make several requests and ensure they only go to healthy backends
	for i := 0; i < 10; i++ {
		backend := algorithm.SelectBackend(backends)
		if backend == nil {
			t.Fatalf("Algorithm returned nil backend")
		}

		// Verify the selected backend is not the failed one
		if backend.Address == fmt.Sprintf("localhost:%d", backendPorts[1]) {
			t.Errorf("Selected failed backend: %s", backend.Address)
		}
	}
}

func TestLoadBalancerHTTPRequests(t *testing.T) {
	// Start mock backend servers
	backendPorts := []int{8087, 8088}
	pool := NewMockServerPool(backendPorts)
	pool.StartAll()
	defer pool.StopAll()

	// Wait for servers to be ready
	if err := pool.WaitForHealthy(5 * time.Second); err != nil {
		t.Fatalf("Mock servers failed to start: %v", err)
	}

	// Reset request counters
	pool.ResetAllCounters()

	// Make HTTP requests directly to test the mock servers
	client := &http.Client{Timeout: 5 * time.Second}
	requestCount := 20

	// Simulate load balancer behavior by alternating requests
	for i := 0; i < requestCount; i++ {
		port := backendPorts[i%len(backendPorts)]
		url := fmt.Sprintf("http://localhost:%d/", port)

		resp, err := client.Get(url)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Request %d: expected status 200, got %d", i, resp.StatusCode)
		}

		// Read and close response body
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// Verify request distribution
	distribution := pool.GetRequestDistribution()
	expectedPerServer := requestCount / len(backendPorts)

	for serverName, count := range distribution {
		if count != int64(expectedPerServer) {
			t.Errorf("Server %s: expected %d requests, got %d",
				serverName, expectedPerServer, count)
		}
	}
}

func TestBackendReachability(t *testing.T) {
	// Test with a server that's not running
	server := &backend.Server{
		Address: "localhost",
		Port:    9999, // Unlikely to be in use
		Healthy: false,
	}

	if server.IsReachable() {
		t.Error("Expected unreachable server to return false")
	}

	// Test with a running server
	backendPorts := []int{8089}
	pool := NewMockServerPool(backendPorts)
	pool.StartAll()
	defer pool.StopAll()

	// Wait for server to be ready
	if err := pool.WaitForHealthy(5 * time.Second); err != nil {
		t.Fatalf("Mock server failed to start: %v", err)
	}

	runningServer := &backend.Server{
		Address: "localhost",
		Port:    8089,
		Healthy: false,
	}

	if !runningServer.IsReachable() {
		t.Error("Expected reachable server to return true")
	}
}

func TestConcurrentRequests(t *testing.T) {
	// Start mock backend servers
	backendPorts := []int{8090, 8091, 8092}
	pool := NewMockServerPool(backendPorts)
	pool.StartAll()
	defer pool.StopAll()

	// Wait for servers to be ready
	if err := pool.WaitForHealthy(5 * time.Second); err != nil {
		t.Fatalf("Mock servers failed to start: %v", err)
	}

	// Reset counters
	pool.ResetAllCounters()

	// Create backends for algorithm testing
	backends := make([]balancer.Backend, len(backendPorts))
	for i, port := range backendPorts {
		backends[i] = balancer.Backend{
			Address: fmt.Sprintf("localhost:%d", port),
			Healthy: true,
		}
	}

	algorithm := balancer.NewRoundRobinAlgorithm()

	// Simulate concurrent requests
	const numGoroutines = 10
	const requestsPerGoroutine = 5

	results := make(chan string, numGoroutines*requestsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < requestsPerGoroutine; j++ {
				backend := algorithm.SelectBackend(backends)
				if backend != nil {
					results <- backend.Address
				}
			}
		}()
	}

	// Collect results
	selections := make(map[string]int)
	for i := 0; i < numGoroutines*requestsPerGoroutine; i++ {
		select {
		case addr := <-results:
			selections[addr]++
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}

	// Verify roughly equal distribution
	expectedPerBackend := (numGoroutines * requestsPerGoroutine) / len(backends)
	tolerance := expectedPerBackend / 2 // 50% tolerance for concurrent access

	for _, backend := range backends {
		count := selections[backend.Address]
		if count < expectedPerBackend-tolerance || count > expectedPerBackend+tolerance {
			t.Errorf("Backend %s: expected ~%d selections, got %d",
				backend.Address, expectedPerBackend, count)
		}
	}
}
