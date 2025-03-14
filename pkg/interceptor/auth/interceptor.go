package auth

import (
	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type Interceptor struct {
	interceptor.NoOpInterceptor
}

func (auth *Interceptor) BindSocketConnection(connection *websocket.Conn) error {
	return nil
}

func (auth *Interceptor) Close() error {
	return nil
}
