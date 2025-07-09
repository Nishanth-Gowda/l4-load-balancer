# L4 Load Balancer

A high-performance Layer 4 (TCP) load balancer implemented in Go.

## Features

- **Multiple Load Balancing Algorithms**
  - Round Robin
  - Least Connections (TODO)
  
- **Health Checking**
  - Automatic backend health monitoring
  - Configurable check intervals
  - Unhealthy backend removal

- **Connection Pooling**
  - Efficient connection reuse
  - Configurable pool sizes
  - Connection lifecycle management

- **Configuration Management**
  - YAML-based configuration
  - Runtime configuration reloading (TODO)

## Project Structure

```
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── balancer/
│   │   ├── balancer.go         # Core load balancer logic
│   │   └── algorithms.go       # Load balancing algorithms
│   ├── backend/
│   │   └── backend.go          # Backend server management
│   ├── health/
│   │   └── checker.go          # Health checking functionality
│   └── config/
│       └── config.go           # Configuration management
├── pkg/
│   └── pool/
│       └── pool.go             # Connection pooling
├── configs/
│   └── config.yaml             # Sample configuration
├── go.mod
├── go.sum
└── README.md
```

## Configuration

The load balancer is configured using a YAML file. See `configs/config.yaml` for an example.

### Configuration Options

- `loadbalancer.listen_address`: Address to listen on (e.g., ":8080")
- `loadbalancer.algorithm`: Load balancing algorithm ("round_robin", "least_connections")
- `backends`: List of backend servers with address and port
- `healthcheck.interval`: How often to check backend health
- `healthcheck.timeout`: Timeout for health checks

## Usage

1. **Build the application:**
   ```bash
   go build -o l4-load-balancer ./cmd
   ```

2. **Run with default configuration:**
   ```bash
   ./l4-load-balancer
   ```

3. **Run with custom configuration:**
   ```bash
   ./l4-load-balancer -config configs/config.yaml
   ```

## Development

### Prerequisites

- Go 1.19 or later

### Running Tests

```bash
go test ./...
```

### Adding New Load Balancing Algorithms

1. Implement the `Algorithm` interface in `internal/balancer/algorithms.go`
2. Add the algorithm to the algorithm factory/registry
3. Update configuration options

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License (add your license details)

## TODO

- [ ] Implement least connections algorithm
- [ ] Add weighted round robin
- [ ] Add SSL/TLS termination
- [ ] Add metrics and monitoring
- [ ] Add graceful shutdown
- [ ] Add configuration hot-reloading
- [ ] Add rate limiting
- [ ] Add connection limiting per backend
- [ ] Add logging configuration
- [ ] Add Docker support 