package auth

import (
	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type Interceptor struct {
	interceptor.NoOpInterceptor
}

func (auth *Interceptor) BindConnection(connection *websocket.Conn) {
	return
}

func (auth *Interceptor) Close() error {
	return nil
}
