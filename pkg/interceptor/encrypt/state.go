package encrypt

import (
	"context"
	"time"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type stats struct {
}

type state struct {
	stats
	peerID    string
	privKey   PrivateKey // THIS private key (not the peers')
	salt      Salt
	encryptor encryptor
	initDone  chan struct{}
	writer    interceptor.Writer
	reader    interceptor.Reader
	cancel    context.CancelFunc
	ctx       context.Context
}

func (state *state) waitUntilInit() error {
	timout, cancel := context.WithTimeout(state.ctx, 5*time.Second)
	defer cancel()

	select {
	case <-state.initDone:
		return nil
	case <-timout.Done():
		return timout.Err()
	}
}
