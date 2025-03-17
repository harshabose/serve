package ping

import (
	"errors"
	"sync"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

// manager centrally coordinates ping/pong state tracking across multiple
// websocket connections. It maintains a thread-safe registry of connection-specific
// state and enforces configuration limits like maximum history size.
// This central component delegates processing to individual states while
// providing synchronized access to them.
type manager struct {
	states map[interceptor.Connection]*state // Map of connection-specific ping/pong states
	max    uint16                            // Maximum number of ping/pong records to keep per connection
	mux    sync.RWMutex                      // Mutex for thread-safe access to the states map
}

// createManager constructs a new manager with an empty state map.
// This factory function initializes the core data structure but does not
// set any configuration values like max history size, which should be
// configured through the interceptor options.
//
// Returns:
//   - A new manager ready to track ping/pong states
func createManager() *manager {
	return &manager{
		states: make(map[interceptor.Connection]*state),
	}
}

// manage initializes ping/pong tracking state for a new connection.
// It ensures that each connection has only one state entry and configures
// the new state with the manager's settings like maximum history size.
//
// Parameters:
//   - connection: The websocket connection to create state for
//
// Returns:
//   - An error if state already exists for this connection, nil otherwise
func (manager *manager) manage(connection interceptor.Connection) error {
	_, exists := manager.states[connection]
	if exists {
		return errors.New("ping-pong already exists")
	}

	manager.states[connection] = &state{
		pings: make([]*ping, 0),
		pongs: make([]*pong, 0),
		max:   manager.max,
	}

	return nil
}

// unmanage removes ping/pong tracking state for a connection and performs cleanup.
// It first looks up the connection's state in the registry, then:
// - If found: Calls cleanup on the state and removes it from the registry
// - If not found: Returns an error indicating the connection doesn't exist
//
// Parameters:
//   - connection: The websocket connection to remove state for
//
// Returns:
//   - An error if the connection's state doesn't exist
func (manager *manager) unmanage(connection interceptor.Connection) error {
	if state, exists := manager.states[connection]; exists {
		state.cleanup()
		delete(manager.states, connection)
		return nil
	}
	return errors.New("connection does not exists")
}

// Process handles incoming ping/pong messages by delegating to the specific
// payload type's Process method. This provides polymorphic processing where
// Ping and Pong messages can be handled differently while using a unified interface.
//
// Parameters:
//   - msg: The ping/pong message to process
//   - connection: The websocket connection the message was received on
//
// Returns:
//   - Any error encountered during processing
func (manager *manager) Process(msg *Message, connection interceptor.Connection) error {
	return msg.Payload.Process(manager, connection)
}

// Process implements the Payload.Process method for Pong messages.
// It validates the pong message, finds the associated connection state,
// and records the pong in that state for RTT calculation and statistics.
//
// Parameters:
//   - manager: The ping/pong manager to use for state lookup
//   - connection: The websocket connection the pong was received on
//
// Returns:
//   - Error if validation fails or no state exists for the connection
func (payload *Pong) Process(manager *manager, connection interceptor.Connection) error {
	if err := payload.Validate(); err != nil {
		return err
	}

	manager.mux.Lock()
	defer manager.mux.Unlock()

	state, exists := manager.states[connection]
	if !exists {
		return errors.New("no ping-pong-er exists")
	}
	state.recordPong(payload)
	// SEND PING HERE

	return nil
}

// Process implements the Payload.Process method for Ping messages.
// It validates the ping message, finds the associated connection state,
// and records the ping in that state for tracking and statistics.
// This is typically used to track pings sent by the local endpoint, but
// could also process pings from remote endpoints.
//
// Parameters:
//   - manager: The ping/pong manager to use for state lookup
//   - connection: The websocket connection the ping was received on
//
// Returns:
//   - Error if validation fails or no state exists for the connection
func (payload *Ping) Process(manager *manager, connection interceptor.Connection) error {
	if err := payload.Validate(); err != nil {
		return err
	}

	manager.mux.Lock()
	defer manager.mux.Unlock()

	state, exists := manager.states[connection]
	if !exists {
		return errors.New("owner does not exists")
	}

	state.recordPing(payload)
	// SEND PONG IDEALLY

	return nil
}

// cleanup performs a complete cleanup of all connection states.
// It first calls cleanup on each individual state to allow state-specific
// resource cleanup, then removes the state from the manager's registry.
// This method is typically called during interceptor shutdown.
func (manager *manager) cleanup() {
	manager.mux.Lock()
	defer manager.mux.Unlock()

	for connection, state := range manager.states {
		state.cleanup()
		delete(manager.states, connection)
	}
}

// TODO: Add health management
// Future enhancements could include:
// - Connection health monitoring based on RTT and success rate
// - Escalating ping frequency for connections with degrading health
// - Health status notifications to higher-level components
