package encrypt

import (
	"context"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type stats struct {
}

type state struct {
	stats
	privKey   []byte
	pubKey    []byte
	salt      []byte
	sessionID []byte
	peerID    string
	encryptor encryptor
	writer    interceptor.Writer
	reader    interceptor.Reader
	cancel    context.CancelFunc
	ctx       context.Context
}
