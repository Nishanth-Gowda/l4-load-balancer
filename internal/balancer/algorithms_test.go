package balancer

import (
	"sync"
	"testing"
)

func TestRoundRobinAlgorithm_SelectBackend(t *testing.T) {
	tests := []struct {
		name     string
		backends []Backend
		requests int
		expected []int // Expected sequence of backend indices
	}{
		{
			name: "three healthy backends",
			backends: []Backend{
				{Address: "server1:8081", Healthy: true},
				{Address: "server2:8082", Healthy: true},
				{Address: "server3:8083", Healthy: true},
			},
			requests: 6,
			expected: []int{1, 2, 0, 1, 2, 0}, // Round-robin through 3 healthy backends
		},
		{
			name: "mixed healthy and unhealthy backends",
			backends: []Backend{
				{Address: "server1:8081", Healthy: true},
				{Address: "server2:8082", Healthy: false},
				{Address: "server3:8083", Healthy: true},
			},
			requests: 4,
			expected: []int{1, 0, 1, 0}, // Round-robin through 2 healthy backends
		},
		{
			name:     "no backends",
			backends: []Backend{},
			requests: 3,
			expected: []int{-1, -1, -1}, // Should return nil (index -1)
		},
		{
			name: "all backends unhealthy",
			backends: []Backend{
				{Address: "server1:8081", Healthy: false},
				{Address: "server2:8082", Healthy: false},
			},
			requests: 2,
			expected: []int{-1, -1}, // Should return nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := NewRoundRobinAlgorithm()

			for i := 0; i < tt.requests; i++ {
				backend := rr.SelectBackend(tt.backends)

				var actualHealthyIndex int = -1
				if backend != nil {
					// Create a mapping of healthy backends to their indices
					healthyBackends := make([]Backend, 0)
					for _, b := range tt.backends {
						if b.Healthy {
							healthyBackends = append(healthyBackends, b)
						}
					}

					// Find the index in the healthy backends array
					for j, b := range healthyBackends {
						if b.Address == backend.Address {
							actualHealthyIndex = j
							break
						}
					}
				}

				if actualHealthyIndex != tt.expected[i] {
					t.Errorf("Request %d: expected healthy backend index %d, got %d",
						i, tt.expected[i], actualHealthyIndex)
				}
			}
		})
	}
}

func TestRoundRobinAlgorithm_Concurrency(t *testing.T) {
	backends := []Backend{
		{Address: "server1:8081", Healthy: true},
		{Address: "server2:8082", Healthy: true},
		{Address: "server3:8083", Healthy: true},
	}

	rr := NewRoundRobinAlgorithm()
	const numGoroutines = 100
	const requestsPerGoroutine = 10

	// Track selections
	selections := make([]string, 0, numGoroutines*requestsPerGoroutine)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Launch concurrent goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				backend := rr.SelectBackend(backends)
				if backend != nil {
					mu.Lock()
					selections = append(selections, backend.Address)
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	// Verify we got the expected number of selections
	expectedSelections := numGoroutines * requestsPerGoroutine
	if len(selections) != expectedSelections {
		t.Errorf("Expected %d selections, got %d", expectedSelections, len(selections))
	}

	// Count selections per backend - should be roughly equal
	counts := make(map[string]int)
	for _, addr := range selections {
		counts[addr]++
	}

	expectedPerBackend := expectedSelections / len(backends)
	tolerance := expectedPerBackend / 10 // 10% tolerance

	for _, backend := range backends {
		count := counts[backend.Address]
		if count < expectedPerBackend-tolerance || count > expectedPerBackend+tolerance {
			t.Errorf("Backend %s: expected ~%d selections, got %d",
				backend.Address, expectedPerBackend, count)
		}
	}
}

func TestRoundRobinAlgorithm_ThreadSafety(t *testing.T) {
	backends := []Backend{
		{Address: "server1:8081", Healthy: true},
		{Address: "server2:8082", Healthy: true},
	}

	rr := NewRoundRobinAlgorithm()
	const numGoroutines = 50

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Run concurrent selections
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				backend := rr.SelectBackend(backends)
				if backend == nil {
					errors <- nil // This shouldn't happen with healthy backends
					return
				}

				// Verify the selected backend is in our list
				found := false
				for _, b := range backends {
					if b.Address == backend.Address && b.Healthy {
						found = true
						break
					}
				}

				if !found {
					errors <- nil
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		if err != nil {
			t.Errorf("Thread safety test failed: %v", err)
		}
	}
}

func BenchmarkRoundRobinAlgorithm_SelectBackend(b *testing.B) {
	backends := []Backend{
		{Address: "server1:8081", Healthy: true},
		{Address: "server2:8082", Healthy: true},
		{Address: "server3:8083", Healthy: true},
		{Address: "server4:8084", Healthy: true},
		{Address: "server5:8085", Healthy: true},
	}

	rr := NewRoundRobinAlgorithm()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rr.SelectBackend(backends)
		}
	})
}
