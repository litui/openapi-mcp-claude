package server

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewConnectionManager(t *testing.T) {
	cm := NewConnectionManager()
	assert.NotNil(t, cm)
	assert.NotNil(t, cm.connections)
	assert.Equal(t, 0, cm.GetConnectionCount())
}

func TestConnectionManager_NewConnection(t *testing.T) {
	cm := NewConnectionManager()
	connID := "test-conn-1"

	conn := cm.NewConnection(connID)

	assert.NotNil(t, conn)
	assert.Equal(t, connID, conn.ID)
	assert.Equal(t, StateConnected, conn.State)
	assert.NotNil(t, conn.Channel)
	assert.False(t, conn.CreatedAt.IsZero())
	assert.Nil(t, conn.InitializedAt)

	// Verify it's in the manager
	assert.Equal(t, 1, cm.GetConnectionCount())
	retrieved := cm.GetConnection(connID)
	assert.Equal(t, conn, retrieved)
}

func TestConnectionManager_GetConnection(t *testing.T) {
	cm := NewConnectionManager()
	connID := "test-conn-1"

	// Non-existent connection should return nil
	conn := cm.GetConnection(connID)
	assert.Nil(t, conn)

	// Create and retrieve
	created := cm.NewConnection(connID)
	retrieved := cm.GetConnection(connID)
	assert.Equal(t, created, retrieved)
}

func TestConnectionManager_UpdateState(t *testing.T) {
	cm := NewConnectionManager()
	connID := "test-conn-1"

	// Update non-existent connection should return false
	updated := cm.UpdateState(connID, StateReady)
	assert.False(t, updated)

	// Create connection and update state
	conn := cm.NewConnection(connID)
	assert.Equal(t, StateConnected, conn.State)
	assert.Nil(t, conn.InitializedAt)

	// Update to Initializing
	updated = cm.UpdateState(connID, StateInitializing)
	assert.True(t, updated)
	assert.Equal(t, StateInitializing, conn.State)
	assert.Nil(t, conn.InitializedAt)

	// Update to Ready (should set InitializedAt)
	updated = cm.UpdateState(connID, StateReady)
	assert.True(t, updated)
	assert.Equal(t, StateReady, conn.State)
	assert.NotNil(t, conn.InitializedAt)
	assert.False(t, conn.InitializedAt.IsZero())

	// Update to Ready again (should not change InitializedAt)
	firstInitTime := *conn.InitializedAt
	time.Sleep(1 * time.Millisecond) // Ensure time difference
	updated = cm.UpdateState(connID, StateReady)
	assert.True(t, updated)
	assert.Equal(t, StateReady, conn.State)
	assert.Equal(t, firstInitTime, *conn.InitializedAt)

	// Update to Shutdown
	updated = cm.UpdateState(connID, StateShutdown)
	assert.True(t, updated)
	assert.Equal(t, StateShutdown, conn.State)
}

func TestConnectionManager_RemoveConnection(t *testing.T) {
	cm := NewConnectionManager()
	connID := "test-conn-1"

	// Remove non-existent connection should return false
	removed := cm.RemoveConnection(connID)
	assert.False(t, removed)

	// Create connection and remove
	conn := cm.NewConnection(connID)
	assert.Equal(t, 1, cm.GetConnectionCount())

	removed = cm.RemoveConnection(connID)
	assert.True(t, removed)
	assert.Equal(t, 0, cm.GetConnectionCount())

	// Verify connection is gone
	retrieved := cm.GetConnection(connID)
	assert.Nil(t, retrieved)

	// Verify channel is closed
	select {
	case _, ok := <-conn.Channel:
		assert.False(t, ok, "Channel should be closed")
	default:
		// Channel might be closed but no data to read
		// Try to send to verify it's closed
		defer func() {
			if r := recover(); r != nil {
				// Expected: sending on closed channel causes panic
			}
		}()
		// This should panic if channel is closed
		select {
		case conn.Channel <- jsonRPCResponse{}:
			t.Fatal("Should not be able to send to closed channel")
		default:
			// Channel is closed and full, which is expected
		}
	}
}

func TestConnectionManager_GetConnectionsByState(t *testing.T) {
	cm := NewConnectionManager()

	// Create connections in different states
	conn1 := cm.NewConnection("conn1")
	conn2 := cm.NewConnection("conn2")
	conn3 := cm.NewConnection("conn3")

	cm.UpdateState("conn2", StateInitializing)
	cm.UpdateState("conn3", StateReady)

	// Get connections by state
	connected := cm.GetConnectionsByState(StateConnected)
	assert.Len(t, connected, 1)
	assert.Equal(t, conn1.ID, connected[0].ID)

	initializing := cm.GetConnectionsByState(StateInitializing)
	assert.Len(t, initializing, 1)
	assert.Equal(t, conn2.ID, initializing[0].ID)

	ready := cm.GetConnectionsByState(StateReady)
	assert.Len(t, ready, 1)
	assert.Equal(t, conn3.ID, ready[0].ID)

	shutdown := cm.GetConnectionsByState(StateShutdown)
	assert.Len(t, shutdown, 0)
}

func TestConnectionManager_ConcurrentAccess(t *testing.T) {
	cm := NewConnectionManager()
	const numGoroutines = 10
	const connectionsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent connection creation
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < connectionsPerGoroutine; j++ {
				connID := fmt.Sprintf("conn-%d-%d", routineID, j)
				conn := cm.NewConnection(connID)
				assert.NotNil(t, conn)
				assert.Equal(t, connID, conn.ID)
			}
		}(i)
	}

	wg.Wait()

	// Verify all connections were created
	expectedCount := numGoroutines * connectionsPerGoroutine
	assert.Equal(t, expectedCount, cm.GetConnectionCount())

	// Concurrent state updates and removals
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < connectionsPerGoroutine; j++ {
				connID := fmt.Sprintf("conn-%d-%d", routineID, j)
				
				// Update state
				updated := cm.UpdateState(connID, StateReady)
				assert.True(t, updated)
				
				// Remove connection
				removed := cm.RemoveConnection(connID)
				assert.True(t, removed)
			}
		}(i)
	}

	wg.Wait()

	// Verify all connections were removed
	assert.Equal(t, 0, cm.GetConnectionCount())
}

func TestConnectionState_String(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected string
	}{
		{StateConnected, "Connected"},
		{StateInitializing, "Initializing"},
		{StateReady, "Ready"},
		{StateShutdown, "Shutdown"},
		{ConnectionState(999), "Unknown"},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.expected, tc.state.String())
	}
}