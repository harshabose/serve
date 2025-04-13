package encrypt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

// Common state errors
var (
	ErrInitializationTimeout = errors.New("encryption initialization timed out")
	ErrContextCanceled       = errors.New("operation canceled")
)

type stats struct {
}

// state maintains the connection-specific encryption state
type state struct {
	stats
	peerID    string
	privKey   PrivateKey // THIS private key (not the peers')
	salt      Salt       // Salt used for key derivation
	encryptor Encryptor  // Encryption implementation
	initDone  chan struct{}
	writer    interceptor.Writer
	reader    interceptor.Reader
	cancel    context.CancelFunc
	ctx       context.Context
}

// waitUntilInit blocks until encryption is initialized or times out
func (state *state) waitUntilInit() error {
	// Create a timeout context
	timeout, cancel := context.WithTimeout(state.ctx, 5*time.Second)
	defer cancel()

	select {
	case <-state.initDone:
		// Encryption successfully initialized
		return nil
	case <-timeout.Done():
		if errors.Is(timeout.Err(), context.DeadlineExceeded) {
			return ErrInitializationTimeout
		}
		return fmt.Errorf("initialization interrupted: %w", timeout.Err())
	case <-state.ctx.Done():
		// Parent context canceled
		return ErrContextCanceled
	}
}
