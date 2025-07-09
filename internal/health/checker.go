package health

import (
	"log"
	"time"

	"l4-load-balancer/internal/backend"
)

// Checker performs health checks on backend servers
type Checker struct {
	manager  *backend.Manager
	interval time.Duration
	timeout  time.Duration
	stopCh   chan struct{}
}

// NewChecker creates a new health checker
func NewChecker(manager *backend.Manager, interval, timeout time.Duration) *Checker {
	return &Checker{
		manager:  manager,
		interval: interval,
		timeout:  timeout,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the health checking process
func (c *Checker) Start() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	// Initial health check
	c.checkAll()

	for {
		select {
		case <-ticker.C:
			c.checkAll()
		case <-c.stopCh:
			log.Println("Health checker stopped")
			return
		}
	}
}

// Stop stops the health checker
func (c *Checker) Stop() {
	close(c.stopCh)
}

// checkAll performs health checks on all servers
func (c *Checker) checkAll() {
	servers := c.manager.GetAllServers()

	for _, server := range servers {
		c.checkServer(server)
	}
}

// checkServer performs a health check on a single server
func (c *Checker) checkServer(server *backend.Server) {
	server.LastChecked = time.Now()

	if server.IsReachable() {
		if !server.Healthy {
			log.Printf("Server %s is now healthy", server.GetAddress())
		}
		server.Healthy = true
	} else {
		if server.Healthy {
			log.Printf("Server %s is now unhealthy", server.GetAddress())
		}
		server.Healthy = false
	}
}
