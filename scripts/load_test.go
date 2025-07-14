package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	targetURL         = flag.String("url", "http://localhost:8080", "Target URL for load testing")
	numWorkers        = flag.Int("workers", 10, "Number of concurrent workers")
	requestsPerWorker = flag.Int("requests", 100, "Requests per worker")
	duration          = flag.Duration("duration", 0, "Test duration (0 means use request count)")
)

type LoadTestResults struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalDuration   time.Duration
	RequestsPerSec  float64
	MinLatency      time.Duration
	MaxLatency      time.Duration
	AvgLatency      time.Duration
}

type LatencyTracker struct {
	mu        sync.Mutex
	latencies []time.Duration
}

func (lt *LatencyTracker) AddLatency(latency time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	lt.latencies = append(lt.latencies, latency)
}

func (lt *LatencyTracker) GetStats() (min, max, avg time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if len(lt.latencies) == 0 {
		return 0, 0, 0
	}

	min = lt.latencies[0]
	max = lt.latencies[0]
	var total time.Duration

	for _, latency := range lt.latencies {
		if latency < min {
			min = latency
		}
		if latency > max {
			max = latency
		}
		total += latency
	}

	avg = total / time.Duration(len(lt.latencies))
	return min, max, avg
}

func worker(id int, results *LoadTestResults, tracker *LatencyTracker, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	var requests int
	if *duration > 0 {
		// Time-based testing
		deadline := time.Now().Add(*duration)
		for time.Now().Before(deadline) {
			performRequest(client, results, tracker)
			requests++
		}
	} else {
		// Request count-based testing
		for i := 0; i < *requestsPerWorker; i++ {
			performRequest(client, results, tracker)
			requests++
		}
	}

	log.Printf("Worker %d completed %d requests", id, requests)
}

func performRequest(client *http.Client, results *LoadTestResults, tracker *LatencyTracker) {
	start := time.Now()

	resp, err := client.Get(*targetURL)
	latency := time.Since(start)

	atomic.AddInt64(&results.TotalRequests, 1)
	tracker.AddLatency(latency)

	if err != nil {
		atomic.AddInt64(&results.FailedRequests, 1)
		log.Printf("Request failed: %v", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		atomic.AddInt64(&results.SuccessRequests, 1)
		// Read the response body to ensure complete processing
		io.Copy(io.Discard, resp.Body)
	} else {
		atomic.AddInt64(&results.FailedRequests, 1)
		log.Printf("Request failed with status: %d", resp.StatusCode)
	}
}

func main() {
	flag.Parse()

	log.Printf("Starting load test against %s", *targetURL)
	log.Printf("Workers: %d", *numWorkers)

	if *duration > 0 {
		log.Printf("Duration: %v", *duration)
	} else {
		log.Printf("Requests per worker: %d", *requestsPerWorker)
		log.Printf("Total requests: %d", *numWorkers**requestsPerWorker)
	}

	results := &LoadTestResults{}
	tracker := &LatencyTracker{}

	var wg sync.WaitGroup
	start := time.Now()

	// Start workers
	for i := 0; i < *numWorkers; i++ {
		wg.Add(1)
		go worker(i, results, tracker, &wg)
	}

	// Wait for all workers to complete
	wg.Wait()

	results.TotalDuration = time.Since(start)
	results.RequestsPerSec = float64(results.TotalRequests) / results.TotalDuration.Seconds()
	results.MinLatency, results.MaxLatency, results.AvgLatency = tracker.GetStats()

	// Print results
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("LOAD TEST RESULTS")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Total Requests:    %d\n", results.TotalRequests)
	fmt.Printf("Successful:        %d\n", results.SuccessRequests)
	fmt.Printf("Failed:            %d\n", results.FailedRequests)
	fmt.Printf("Success Rate:      %.2f%%\n", float64(results.SuccessRequests)/float64(results.TotalRequests)*100)
	fmt.Printf("Total Duration:    %v\n", results.TotalDuration)
	fmt.Printf("Requests/sec:      %.2f\n", results.RequestsPerSec)
	fmt.Println("\nLatency Statistics:")
	fmt.Printf("  Min:             %v\n", results.MinLatency)
	fmt.Printf("  Max:             %v\n", results.MaxLatency)
	fmt.Printf("  Average:         %v\n", results.AvgLatency)
	fmt.Println(strings.Repeat("=", 50))

	if results.FailedRequests > 0 {
		fmt.Printf("\nWARNING: %d requests failed\n", results.FailedRequests)
	}
}
