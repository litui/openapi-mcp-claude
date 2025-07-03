package server

import (
	"sync"
	"time"
)

// ConnectionState represents the state of an MCP connection
type ConnectionState int

const (
	StateConnected ConnectionState = iota
	StateInitializing
	StateReady
	StateShutdown
)

// String returns the string representation of the connection state
func (s ConnectionState) String() string {
	switch s {
	case StateConnected:
		return "Connected"
	case StateInitializing:
		return "Initializing"
	case StateReady:
		return "Ready"
	case StateShutdown:
		return "Shutdown"
	default:
		return "Unknown"
	}
}

// Connection represents an MCP connection
type Connection struct {
	ID            string
	State         ConnectionState
	Channel       chan jsonRPCResponse
	InitializedAt *time.Time
	CreatedAt     time.Time
}

// ConnectionManager manages MCP connections and their states
type ConnectionManager struct {
	connections map[string]*Connection
	mutex       sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*Connection),
	}
}

// NewConnection creates a new connection with the given ID
func (cm *ConnectionManager) NewConnection(id string) *Connection {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	conn := &Connection{
		ID:        id,
		State:     StateConnected,
		Channel:   make(chan jsonRPCResponse, messageChannelBufferSize),
		CreatedAt: time.Now(),
	}
	cm.connections[id] = conn
	return conn
}

// GetConnection retrieves a connection by ID
func (cm *ConnectionManager) GetConnection(id string) *Connection {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.connections[id]
}

// UpdateState updates the state of a connection
func (cm *ConnectionManager) UpdateState(id string, state ConnectionState) bool {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	conn, exists := cm.connections[id]
	if !exists {
		return false
	}

	oldState := conn.State
	conn.State = state

	// Set initialized timestamp when moving to Ready state
	if state == StateReady && oldState != StateReady {
		now := time.Now()
		conn.InitializedAt = &now
	}

	return true
}

// RemoveConnection removes a connection from the manager
func (cm *ConnectionManager) RemoveConnection(id string) bool {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	conn, exists := cm.connections[id]
	if !exists {
		return false
	}

	// Close the channel if it's not already closed
	select {
	case <-conn.Channel:
		// Channel is already closed
	default:
		close(conn.Channel)
	}

	delete(cm.connections, id)
	return true
}

// GetConnectionCount returns the total number of active connections
func (cm *ConnectionManager) GetConnectionCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return len(cm.connections)
}

// GetConnectionsByState returns connections in a specific state
func (cm *ConnectionManager) GetConnectionsByState(state ConnectionState) []*Connection {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var connections []*Connection
	for _, conn := range cm.connections {
		if conn.State == state {
			connections = append(connections, conn)
		}
	}
	return connections
}