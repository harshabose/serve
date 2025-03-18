package pong

import (
	"context"
	"sync"
	"time"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

// pong represents a single pong response record.
// It stores information about a received pong message including its unique ID,
// the calculated round-trip time, and when it was received. This data is used
// for connection health analysis and statistics.
type pong struct {
	messageid string    // Unique identifier matching the corresponding ping
	timestamp time.Time // When this pong was received
}

// ping represents a single ping request record.
// It stores information about an already sent ping message including its unique ID
// and when it was sent. This allows for matching with corresponding pongs
// and calculating accurate round-trip times.
type ping struct {
	messageid string    // Unique identifier for matching with corresponding pong
	timestamp time.Time // When this ping was sent
}

// recent tracks the most recently processed ping and pong messages.
// This provides quick access to the latest connection health data
// without needing to search through the full history arrays.
type recent struct {
	ping *ping // Most recent ping sent
	pong *pong // Most recent pong received
}

// state maintains connection-specific ping/pong tracking information.
// Each websocket connection has its own state instance that records
// ping/pong history, calculates statistics, and provides methods for
// analyzing connection health and performance.
type state struct {
	peerid string
	writer interceptor.Writer
	reader interceptor.Reader
	pongs  []*pong      // Historical record of pongs received
	pings  []*ping      // Historical record of pings sent
	max    uint16       // Maximum number of ping/pong records to keep
	recvd  int          // Total count of pongs received
	sent   int          // Total count of pings sent
	recent recent       // Most recent ping and pong
	mux    sync.RWMutex // Mutex for thread-safe access to state
	ctx    context.Context
	cancel context.CancelFunc
}

// recordPong processes a received pong message and updates the state accordingly.
// It calculates the round-trip time based on the original ping timestamp,
// records the pong in the history (maintaining the maximum history size),
// updates the recent pong reference, and increments the received count.
//
// Parameters:
//   - payload: The pong message received from the client
func (state *state) recordPong(payload *Pong) {
	state.mux.Lock()
	defer state.mux.Unlock()

	pong := &pong{
		messageid: payload.MessageID,
		timestamp: time.Now(),
	}
	state.recent.pong = pong

	if uint16(len(state.pongs)) >= state.max {
		if len(state.pongs) > 0 {
			state.pongs = state.pongs[1:]
		}
	}
	state.pongs = append(state.pongs, pong)
	state.recvd++
}

// recordPing processes an already sent ping message and updates the state accordingly.
// It records the ping in the history (maintaining the maximum history size),
// updates the recent ping reference, and increments the already sent count.
// This is typically called when the interceptor sends a ping, but could also
// track pings from the client in bidirectional ping/pong implementations.
//
// Parameters:
//   - payload: The ping message sent to the client
func (state *state) recordPing(payload *Ping) {
	state.mux.Lock()
	defer state.mux.Unlock()

	ping := &ping{
		messageid: payload.MessageID,
		timestamp: payload.Timestamp,
	}
	state.recent.ping = ping

	if uint16(len(state.pings)) >= state.max {
		if len(state.pings) > 0 {
			state.pings = state.pings[1:]
		}
	}
	state.pings = append(state.pings, ping)
	state.sent++
}

// GetSuccessRate returns the percentage of pings that received corresponding pongs.
// This metric helps assess connection reliability by measuring how many ping
// requests are successfully acknowledged by the client.
//
// Returns:
//   - The success rate as a percentage (0-100), or zero if no pings have been sent
func (state *state) GetSuccessRate() float64 {
	state.mux.RLock()
	defer state.mux.RUnlock()

	if state.sent == 0 {
		return 0
	}

	return 100.0 * (1.0 - float64(state.sent-state.recvd)/float64(state.sent))
}

// cleanup releases all resources held by this state.
// It clears all ping and pong records, resets counters, and removes references
// to recent ping/pong objects. This is typically called when a connection
// is closed or when the interceptor is shutting down.
func (state *state) cleanup() {
	state.mux.Lock()
	defer state.mux.Unlock()

	state.pings = nil
	state.pongs = nil
	state.max = 0
	state.sent = 0
	state.recvd = 0
	state.recent.pong = nil
	state.recent.ping = nil
}
