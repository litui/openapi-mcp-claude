package server

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
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
	ID            string               `yaml:"id"`
	State         ConnectionState      `yaml:"state"`
	Channel       chan jsonRPCResponse `yaml:"-"`
	InitializedAt *time.Time           `yaml:"initializedAt"`
	CreatedAt     time.Time            `yaml:"createdAt"`
}

// ConnectionManager manages MCP connections and their states
type ConnectionManager struct {
	connections map[string]*Connection `yaml:"connection"`
	mutex       sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	viper.SetConfigFile("/app/spec/openapi-mcp-state.yaml")

	connections := make(map[string]*Connection)
	// cmBytes, err := yaml.Marshal(connections)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if !viper.IsSet("connection") {
				viper.Set("connection", connections)
				viper.WriteConfig()
			}
		} else {
			viper.Set("connection", connections)
			viper.WriteConfig()
		}
	} else {
		tempCm := viper.GetStringMap("connection")
		for m, c := range tempCm {
			connBytes, _ := yaml.Marshal(c)
			log.Println(string(connBytes))
			tCmc := &Connection{}
			err := yaml.Unmarshal(connBytes, tCmc)
			if err != nil {
				log.Panic(err)
			}
			connections[m] = tCmc
			connections[m].Channel = make(chan jsonRPCResponse, messageChannelBufferSize)
		}
	}

	return &ConnectionManager{
		connections: connections,
	}
}

// NewConnection creates a new connection with the given ID
func (cm *ConnectionManager) NewConnection(id string) *Connection {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	conn := &Connection{
		ID:        strings.ToLower(id),
		State:     StateConnected,
		Channel:   make(chan jsonRPCResponse, messageChannelBufferSize),
		CreatedAt: time.Now(),
	}

	cm.connections[strings.ToLower(id)] = conn
	viper.Set("connection", cm.connections)
	viper.WriteConfig()
	return conn
}

// GetConnection retrieves a connection by ID
func (cm *ConnectionManager) GetConnection(id string) *Connection {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return cm.connections[strings.ToLower(id)]
}

// UpdateState updates the state of a connection
func (cm *ConnectionManager) UpdateState(id string, state ConnectionState) bool {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	conn, ok := cm.connections[strings.ToLower(id)]
	if !ok {
		return false
	}

	oldState := conn.State
	conn.State = state

	// Set initialized timestamp when moving to Ready state
	if state == StateReady && oldState != StateReady {
		now := time.Now()
		conn.InitializedAt = &now
	}

	viper.Set("connection", cm.connections)
	viper.WriteConfig()

	return true
}

// RemoveConnection removes a connection from the manager
func (cm *ConnectionManager) RemoveConnection(id string) bool {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	conn, ok := cm.connections[strings.ToLower(id)]
	if !ok {
		return false
	}

	// Close the channel if it's not already closed
	select {
	case <-conn.Channel:
		// Channel is already closed
	default:
		close(conn.Channel)
	}

	viper.Set("connection", cm.connections)
	viper.WriteConfig()

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
