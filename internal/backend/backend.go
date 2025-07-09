package backend

import (
	"fmt"
	"net"
	"time"
)

// Server represents a backend server
type Server struct {
	Address     string
	Port        int
	Healthy     bool
	LastChecked time.Time
}

// Manager manages backend servers
type Manager struct {
	servers []*Server
}

// NewManager creates a new backend manager
func NewManager() *Manager {
	return &Manager{
		servers: make([]*Server, 0),
	}
}

// AddServer adds a backend server
func (m *Manager) AddServer(address string, port int) {
	server := &Server{
		Address: address,
		Port:    port,
		Healthy: false,
	}
	m.servers = append(m.servers, server)
}

// GetHealthyServers returns all healthy servers
func (m *Manager) GetHealthyServers() []*Server {
	healthy := make([]*Server, 0)
	for _, server := range m.servers {
		if server.Healthy {
			healthy = append(healthy, server)
		}
	}
	return healthy
}

// GetAllServers returns all servers
func (m *Manager) GetAllServers() []*Server {
	return m.servers
}

// GetAddress returns the full address of the server
func (s *Server) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

// IsReachable checks if the server is reachable
func (s *Server) IsReachable() bool {
	conn, err := net.DialTimeout("tcp", s.GetAddress(), 5*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
