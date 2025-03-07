package interceptor

import (
	"io"

	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type Registry struct {
	factories []Factory
}

func (registry *Registry) Register(factory Factory) {
	registry.factories = append(registry.factories, factory)
}

func (registry *Registry) Build(id string) (Interceptor, error) {
	if len(registry.factories) == 0 {
		return &NoInterceptor{}, nil
	}

	interceptors := make([]Interceptor, 0)
	for _, factory := range registry.factories {
		interceptor, err := factory.NewInterceptor(id)
		if err != nil {
			return nil, err
		}

		interceptors = append(interceptors, interceptor)
	}

	return CreateChain(interceptors), nil
}

// Factory provides an interface for constructing interceptors
type Factory interface {
	NewInterceptor(id string) (Interceptor, error)
}

// Interceptor are transformers which bind to incoming, outgoing and connection of a client of the websocket. This can
// be used to add functionalities to the websocket connection.
type Interceptor interface {
	// BindIncoming binds to incoming messages to a client
	BindIncoming(IncomingReader) IncomingReader

	// BindOutgoing binds to outgoing messages from a client
	BindOutgoing(OutgoingWriter) OutgoingWriter

	// BindConnection binds to the websocket connection itself
	BindConnection(Connection) Connection

	io.Closer
}

type IncomingReader interface {
	Read([]byte) (int, error)
}

type OutgoingWriter interface {
	Write(message message.BaseMessage) (int, error)
}

type Connection interface {
}
