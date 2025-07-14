# Testing Guide for L4 Load Balancer

This guide explains how to test the L4 load balancer at different levels to ensure it works correctly and performs well.

## 🧪 Test Types Overview

### 1. **Unit Tests** - Algorithm Correctness
- **Location**: `internal/balancer/algorithms_test.go`
- **Purpose**: Test the round-robin algorithm logic
- **Focus**: Thread safety, distribution fairness, edge cases

### 2. **Integration Tests** - Component Interaction  
- **Location**: `test/integration_test.go`
- **Purpose**: Test how components work together
- **Focus**: Health checking, failover, real server interaction

### 3. **Load Tests** - Performance & Scalability
- **Location**: `scripts/load_test.go`
- **Purpose**: Stress test under concurrent load
- **Focus**: Throughput, latency, resource usage

### 4. **Manual Tests** - End-to-End Verification
- **Location**: `scripts/test_runner.sh`
- **Purpose**: Complete system testing
- **Focus**: Real-world scenarios, user acceptance

## 🚀 Quick Testing Commands

### Run All Tests at Once
```bash
# Complete test suite (recommended)
./scripts/test_runner.sh
```

### Individual Test Types

#### Unit Tests
```bash
# Basic unit tests
go test ./internal/balancer -v

# With benchmarks
go test ./internal/balancer -v -bench=. -benchmem

# Race condition detection
go test ./internal/balancer -v -race
```

#### Integration Tests
```bash
# Full integration suite
go test ./test -v

# Specific test
go test ./test -v -run TestLoadBalancerIntegration
```

#### Load Testing
```bash
# Build load test tool
go build -o load_test ./scripts/load_test.go

# Basic load test
./load_test -url http://localhost:8080 -workers 10 -requests 100

# High-load test  
./load_test -url http://localhost:8080 -workers 50 -requests 500

# Duration-based test
./load_test -url http://localhost:8080 -workers 20 -duration 30s
```

## 📋 Test Scenarios Explained

### 1. Round-Robin Algorithm Tests

```go
// What the test verifies:
index := atomic.AddUint64(&rr.counter, 1) % uint64(len(healthy))
```

**Test Cases:**
- ✅ **Sequential Distribution**: Requests go to backends in order (0→1→2→0...)
- ✅ **Thread Safety**: Multiple goroutines don't cause race conditions  
- ✅ **Healthy-Only Selection**: Unhealthy backends are skipped
- ✅ **Empty Backend Handling**: Gracefully handles no backends
- ✅ **Performance**: Benchmarks selection speed under load

**Key Verification:**
- **Fairness**: Each healthy backend gets roughly equal requests
- **Atomicity**: Counter increments are thread-safe
- **Correctness**: Selection follows round-robin pattern

### 2. Health Check Integration Tests

**Scenarios:**
- ✅ **Initial Health Detection**: All servers start as healthy
- ✅ **Failure Detection**: Health checker detects when servers go down
- ✅ **Automatic Failover**: Load balancer stops sending traffic to failed servers
- ✅ **Recovery Detection**: Servers that come back online are re-included

**Test Flow:**
```
1. Start 3 backend servers
2. Verify all are marked healthy
3. Stop 1 server (simulate failure)
4. Wait for health check cycle
5. Verify only 2 servers receive traffic
6. Confirm failed server is excluded
```

### 3. Load Testing Scenarios

**Metrics Measured:**
- **Throughput**: Requests per second
- **Latency**: Min/Max/Average response times
- **Success Rate**: Percentage of successful requests
- **Distribution**: Request spread across backends

**Test Configurations:**
```bash
# Light load
./load_test -workers 5 -requests 50

# Medium load  
./load_test -workers 25 -requests 200

# Heavy load
./load_test -workers 100 -requests 1000

# Sustained load
./load_test -workers 50 -duration 60s
```

## 🔧 Setting Up Test Environment

### Prerequisites
```bash
# Ensure you have these installed:
- Go 1.19+
- Python 3 (for mock servers)
- curl (for HTTP testing)
- lsof (for port checking)
```

### Manual Backend Servers
```bash
# Start simple HTTP servers for testing
python3 -m http.server 8081 &
python3 -m http.server 8082 &
python3 -m http.server 8083 &

# Test load balancer against these
go run ./cmd/main.go
```

### Using Docker (Optional)
```bash
# Create backend containers
docker run -d -p 8081:80 nginx
docker run -d -p 8082:80 nginx  
docker run -d -p 8083:80 nginx

# Test load balancer
go run ./cmd/main.go
```

## 📊 Test Results Interpretation

### Unit Test Results
```
PASS: TestRoundRobinAlgorithm_SelectBackend
  - Distribution is fair ✓
  - Thread safety maintained ✓
  - Edge cases handled ✓

PASS: TestRoundRobinAlgorithm_Concurrency  
  - No race conditions ✓
  - Performance acceptable ✓
```

### Load Test Results
```
==================================================
LOAD TEST RESULTS
==================================================
Total Requests:    1000
Successful:        1000
Failed:            0
Success Rate:      100.00%
Total Duration:    2.5s
Requests/sec:      400.00

Latency Statistics:
  Min:             1ms
  Max:             50ms  
  Average:         5ms
==================================================
```

**Good Performance Indicators:**
- Success rate > 99%
- Average latency < 10ms
- Requests/sec > 200 (depends on backend)
- Even distribution across backends

**Warning Signs:**
- Failed requests > 1%
- Average latency > 100ms
- Uneven distribution (one backend getting 80%+ traffic)

## 🐛 Troubleshooting Test Issues

### Common Problems

**1. Port Already in Use**
```bash
# Check what's using the port
lsof -i :8080

# Kill process using port
kill -9 $(lsof -t -i:8080)
```

**2. Tests Failing Due to Timing**
```bash
# Increase timeouts in test files
# Or add sleep between operations
sleep 2
```

**3. Race Conditions in Tests**
```bash
# Run with race detector
go test -race ./...
```

**4. Backend Servers Not Starting**
```bash
# Check Python is available
python3 --version

# Check ports are free
netstat -tulpn | grep :808
```

## 🎯 Best Testing Practices

### 1. **Test Pyramid Approach**
- **Many unit tests** (fast, isolated)
- **Some integration tests** (real components)
- **Few end-to-end tests** (full system)

### 2. **Continuous Testing**
```bash
# Run tests on every change
git add . && git commit -m "Changes" && go test ./...
```

### 3. **Performance Baselines**
```bash
# Establish performance baselines
go test -bench=. ./... > baseline.txt

# Compare against baseline
go test -bench=. ./... > current.txt
diff baseline.txt current.txt
```

### 4. **Realistic Test Data**
- Use realistic backend response times
- Test with varying backend health
- Simulate network delays and failures

## 📈 Advanced Testing

### Chaos Engineering
```bash
# Randomly kill backends during load test
while true; do
  sleep $((RANDOM % 30))
  # Kill random backend
  # Restart after delay
done
```

### Memory and CPU Profiling
```bash
# Profile CPU usage
go test -cpuprofile=cpu.prof -bench=. ./internal/balancer

# Profile memory usage  
go test -memprofile=mem.prof -bench=. ./internal/balancer

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof
```

### Integration with CI/CD
```yaml
# Example GitHub Actions workflow
- name: Run Tests
  run: |
    go test ./...
    ./scripts/test_runner.sh
```

## 🏆 Success Criteria

A well-tested load balancer should:
- ✅ Pass all unit tests with 100% success rate
- ✅ Handle 1000+ concurrent requests without failures
- ✅ Maintain < 10ms average latency under normal load
- ✅ Automatically detect and handle backend failures
- ✅ Distribute load evenly across healthy backends
- ✅ Recover gracefully when backends come back online

Remember: **Testing is not just about finding bugs, but also about building confidence in your system's reliability and performance.** 