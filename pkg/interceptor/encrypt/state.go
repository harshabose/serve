package encrypt

import (
	"context"

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
	writer    interceptor.Writer
	reader    interceptor.Reader
	cancel    context.CancelFunc
	ctx       context.Context
}
