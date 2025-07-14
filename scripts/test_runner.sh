#!/bin/bash

# L4 Load Balancer Test Runner
# This script demonstrates how to test the load balancer in different ways

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "=========================================="
echo "L4 Load Balancer Test Suite"
echo "=========================================="

# Function to cleanup background processes
cleanup() {
    echo "Cleaning up background processes..."
    jobs -p | xargs -r kill 2>/dev/null || true
    wait 2>/dev/null || true
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if a port is available
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        return 1  # Port is in use
    else
        return 0  # Port is available
    fi
}

# Function to wait for a service to be ready
wait_for_service() {
    local url=$1
    local timeout=${2:-30}
    local count=0
    
    log "Waiting for service at $url to be ready..."
    
    while [ $count -lt $timeout ]; do
        if curl -s "$url" >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
        ((count++))
    done
    
    return 1
}

# 1. Unit Tests
echo ""
log "Running unit tests..."
cd "$PROJECT_DIR"

if go test ./internal/balancer -v; then
    success "Unit tests passed!"
else
    error "Unit tests failed!"
    exit 1
fi

# 2. Integration Tests
echo ""
log "Running integration tests..."

if go test ./test -v; then
    success "Integration tests passed!"
else
    error "Integration tests failed!"
    exit 1
fi

# 3. Benchmark Tests
echo ""
log "Running benchmark tests..."

go test ./internal/balancer -bench=. -benchmem

# 4. Manual Testing Setup
echo ""
log "Setting up manual test environment..."

# Check if required ports are available
BACKEND_PORTS=(8081 8082 8083)
LB_PORT=8080

for port in "${BACKEND_PORTS[@]}" $LB_PORT; do
    if ! check_port $port; then
        error "Port $port is already in use. Please free it and try again."
        exit 1
    fi
done

# Start mock backend servers
log "Starting mock backend servers..."

for i in "${!BACKEND_PORTS[@]}"; do
    port=${BACKEND_PORTS[$i]}
    server_name="backend-$((i+1))"
    
    # Create a simple HTTP server
    cat > "/tmp/server_${port}.py" << EOF
#!/usr/bin/env python3
import http.server
import socketserver
import json
import threading
import time

class MyHandler(http.server.BaseHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        self.request_count = 0
        super().__init__(*args, **kwargs)
    
    def do_GET(self):
        self.request_count += 1
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        
        response = {
            'server': '$server_name',
            'port': $port,
            'request_count': self.request_count,
            'timestamp': time.time()
        }
        
        self.wfile.write(json.dumps(response).encode())
    
    def log_message(self, format, *args):
        # Suppress default logging
        pass

if __name__ == "__main__":
    with socketserver.TCPServer(("", $port), MyHandler) as httpd:
        print(f"Server $server_name running on port $port")
        httpd.serve_forever()
EOF

    # Start the server in background
    python3 "/tmp/server_${port}.py" &
    echo $! > "/tmp/server_${port}.pid"
    
    log "Started $server_name on port $port"
done

# Wait for backend servers to be ready
sleep 2

for port in "${BACKEND_PORTS[@]}"; do
    if ! wait_for_service "http://localhost:$port" 10; then
        error "Backend server on port $port failed to start"
        exit 1
    fi
done

success "All backend servers are running!"

# 5. Build and start the load balancer
log "Building load balancer..."
if ! go build -o l4-load-balancer ./cmd; then
    error "Failed to build load balancer"
    exit 1
fi

log "Starting load balancer..."
./l4-load-balancer &
LB_PID=$!

# Wait for load balancer to be ready
if ! wait_for_service "http://localhost:$LB_PORT" 10; then
    error "Load balancer failed to start"
    exit 1
fi

success "Load balancer is running on port $LB_PORT"

# 6. Manual Testing
echo ""
log "Running manual tests..."

# Test individual backend servers
log "Testing backend servers directly..."
for port in "${BACKEND_PORTS[@]}"; do
    response=$(curl -s "http://localhost:$port")
    server_name=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin)['server'])")
    log "Backend $port response: $server_name"
done

# Test load balancer distribution
log "Testing load balancer distribution..."
log "Sending 10 requests through load balancer..."

for i in {1..10}; do
    if curl -s "http://localhost:$LB_PORT" >/dev/null; then
        echo -n "."
    else
        echo -n "X"
    fi
done
echo ""

success "Manual tests completed!"

# 7. Load Testing
echo ""
log "Running load tests..."

# Build the load test tool
log "Building load test tool..."
if ! go build -o load_test ./scripts/load_test.go; then
    error "Failed to build load test tool"
    exit 1
fi

# Run load test
log "Running load test with 50 workers, 20 requests each..."
./load_test -url "http://localhost:$LB_PORT" -workers 50 -requests 20

# 8. Health Check Testing
echo ""
log "Testing health check functionality..."

# Stop one backend server to test failover
first_port=${BACKEND_PORTS[0]}
first_pid_file="/tmp/server_${first_port}.pid"

if [ -f "$first_pid_file" ]; then
    first_pid=$(cat "$first_pid_file")
    log "Stopping backend server on port $first_port (PID: $first_pid) to test failover..."
    kill "$first_pid" 2>/dev/null || true
    rm -f "$first_pid_file"
    
    # Wait for health check to detect failure
    sleep 3
    
    log "Testing load balancer with failed backend..."
    for i in {1..5}; do
        curl -s "http://localhost:$LB_PORT" >/dev/null && echo -n "." || echo -n "X"
    done
    echo ""
    
    success "Failover test completed!"
fi

# 9. Performance Analysis
echo ""
log "Performance analysis summary:"
echo "- Unit tests verify algorithm correctness and thread safety"
echo "- Integration tests verify component interaction"
echo "- Load tests measure throughput and latency under stress"
echo "- Health check tests verify automatic failover"

# Final cleanup
echo ""
log "Cleaning up test environment..."

# Kill load balancer
if [ ! -z "$LB_PID" ]; then
    kill "$LB_PID" 2>/dev/null || true
fi

# Kill remaining backend servers
for port in "${BACKEND_PORTS[@]}"; do
    pid_file="/tmp/server_${port}.pid"
    if [ -f "$pid_file" ]; then
        pid=$(cat "$pid_file")
        kill "$pid" 2>/dev/null || true
        rm -f "$pid_file"
    fi
    rm -f "/tmp/server_${port}.py"
done

# Remove built binaries
rm -f l4-load-balancer load_test

success "Test suite completed successfully!"
echo ""
echo "=========================================="
echo "Test Summary:"
echo "✓ Unit tests"
echo "✓ Integration tests" 
echo "✓ Benchmark tests"
echo "✓ Manual testing"
echo "✓ Load testing"
echo "✓ Health check testing"
echo "==========================================" 