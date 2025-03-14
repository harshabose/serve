package room

import (
	"context"
	"sync"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type Interceptor struct {
	interceptor.NoOpInterceptor
	mux sync.Mutex
	ctx context.Context
}

func (room *Interceptor) BindSocketConnection(connection *websocket.Conn) error {

}

func (room *Interceptor) BindSocketWriter(writer interceptor.Writer) interceptor.Writer {

}

func (room *Interceptor) BindSocketReader(reader interceptor.Reader) interceptor.Reader {

}

func (room *Interceptor) UnBindSocketConnection(connection *websocket.Conn) {

}

func (room *Interceptor) UnBindSocketWriter(writer interceptor.Writer) {

}

func (room *Interceptor) UnBindSocketReader(reader interceptor.Reader) {

}

func (room *Interceptor) Close() error {

}
