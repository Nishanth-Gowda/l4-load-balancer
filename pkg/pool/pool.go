package pool

import (
	"net"
	"sync"
	"time"
)

// ConnectionPool manages a pool of connections to backend servers
type ConnectionPool struct {
	address     string
	maxSize     int
	connections chan net.Conn
	mu          sync.RWMutex
	active      int
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(address string, maxSize int) *ConnectionPool {
	return &ConnectionPool{
		address:     address,
		maxSize:     maxSize,
		connections: make(chan net.Conn, maxSize),
	}
}

// Get retrieves a connection from the pool or creates a new one
func (p *ConnectionPool) Get() (net.Conn, error) {
	select {
	case conn := <-p.connections:
		// Test if connection is still alive
		if p.isConnectionValid(conn) {
			return conn, nil
		}
		// Connection is dead, create a new one
		return p.createConnection()
	default:
		// No connections available, create a new one
		return p.createConnection()
	}
}

// Put returns a connection to the pool
func (p *ConnectionPool) Put(conn net.Conn) {
	if conn == nil {
		return
	}

	select {
	case p.connections <- conn:
		// Connection added to pool
	default:
		// Pool is full, close the connection
		conn.Close()
		p.mu.Lock()
		p.active--
		p.mu.Unlock()
	}
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() {
	close(p.connections)
	for conn := range p.connections {
		conn.Close()
	}
}

// createConnection creates a new connection to the backend
func (p *ConnectionPool) createConnection() (net.Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.active >= p.maxSize {
		return nil, ErrPoolExhausted
	}

	conn, err := net.DialTimeout("tcp", p.address, 5*time.Second)
	if err != nil {
		return nil, err
	}

	p.active++
	return conn, nil
}

// isConnectionValid checks if a connection is still valid
func (p *ConnectionPool) isConnectionValid(conn net.Conn) bool {
	// Set a very short deadline to test the connection
	conn.SetReadDeadline(time.Now().Add(time.Millisecond))
	one := make([]byte, 1)
	_, err := conn.Read(one)
	conn.SetReadDeadline(time.Time{}) // Reset deadline

	// If we get EOF or timeout, connection might be closed
	if err != nil {
		conn.Close()
		p.mu.Lock()
		p.active--
		p.mu.Unlock()
		return false
	}

	return true
}

// ActiveConnections returns the number of active connections
func (p *ConnectionPool) ActiveConnections() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.active
}

// PoolSize returns the current size of the connection pool
func (p *ConnectionPool) PoolSize() int {
	return len(p.connections)
}

// Error definitions
var (
	ErrPoolExhausted = &PoolError{"connection pool exhausted"}
)

// PoolError represents a pool-related error
type PoolError struct {
	message string
}

func (e *PoolError) Error() string {
	return e.message
}
