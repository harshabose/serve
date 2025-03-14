package interceptor

import (
	"context"
	"io"

	"github.com/coder/websocket"
)

// Registry maintains a collection of interceptor factories that can be used to
// build a chain of interceptors for a given context and ID.
type Registry struct {
	factories []Factory
}

// Register adds a new interceptor factory to the registry.
// Factories are stored in the order they're registered, which determines
// the order of interceptors in the resulting chain.
func (registry *Registry) Register(factory Factory) {
	registry.factories = append(registry.factories, factory)
}

// Build creates a chain of interceptors by invoking each registered factory.
// If no factories are registered, returns a no-op interceptor.
// The context and ID are passed to each factory to allow for customized
// interceptor creation based on request context or client identity.
func (registry *Registry) Build(ctx context.Context, id string) (Interceptor, error) {
	if len(registry.factories) == 0 {
		return &NoOpInterceptor{}, nil
	}

	interceptors := make([]Interceptor, 0)
	for _, factory := range registry.factories {
		interceptor, err := factory.NewInterceptor(ctx, id)
		if err != nil {
			return nil, err
		}

		interceptors = append(interceptors, interceptor)
	}

	return CreateChain(interceptors), nil
}

// Factory provides an interface for constructing interceptors
type Factory interface {
	NewInterceptor(context.Context, string) (Interceptor, error)
}

// Interceptor defines a transformer that can modify the behavior of websocket connections.
// Interceptors can bind to the connection itself, its writers (for outgoing messages),
// and its readers (for incoming messages). This pattern enables adding functionalities
// like logging, encryption, compression, rate limiting, or analytics to websocket
// connections without modifying the core websocket handling code.
type Interceptor interface {
	// BindSocketConnection is called when a new websocket connection is established.
	// It gives the interceptor an opportunity to set up any connection-specific
	// state or perform initialization tasks for the given connection.
	// Returns an error if the binding process fails, which typically would
	// result in the connection being rejected.
	BindSocketConnection(*websocket.Conn) error

	// BindSocketWriter wraps a writer that handles messages going out to clients.
	// The interceptor receives the original writer and returns a modified writer
	// that adds the interceptor's functionality. For example, an encryption interceptor
	// would return a writer that encrypts messages before passing them to the original writer.
	// The returned writer will be used for all future write operations on the connection.
	BindSocketWriter(Writer) Writer

	// BindSocketReader wraps a reader that handles messages coming in from clients.
	// The interceptor receives the original reader and returns a modified reader
	// that adds the interceptor's functionality. For example, a logging interceptor
	// would return a reader that logs messages after receiving them from the original reader.
	// The returned reader will be used for all future read operations on the connection.
	BindSocketReader(Reader) Reader

	// UnBindSocketConnection is called when a websocket connection is closed or removed.
	// It cleans up any connection-specific resources and state maintained by the interceptor
	// for the given connection, removing it from the collection map to prevent memory leaks.
	UnBindSocketConnection(*websocket.Conn)

	// UnBindSocketWriter is called when a writer is being removed or when the
	// connection is closing. This gives the interceptor an opportunity to clean up
	// any resources or state associated with the writer. The interceptor should
	// release any references to the writer to prevent memory leaks.
	UnBindSocketWriter(Writer)

	// UnBindSocketReader is called when a reader is being removed or when the
	// connection is closing. This gives the interceptor an opportunity to clean up
	// any resources or state associated with the reader. The interceptor should
	// release any references to the reader to prevent memory leaks.
	UnBindSocketReader(Reader)

	// Closer interface implementation for resource cleanup.
	// Close is called when the interceptor itself is being shut down.
	// It should clean up any global resources held by the interceptor.
	io.Closer
}

// Writer is an interface for writing messages to a websocket connection
type Writer interface {
	// Write sends a message to the connection
	// Takes the connection, message type, and data to write
	// Returns any error encountered during writing
	Write(conn *websocket.Conn, messageType websocket.MessageType, data []byte) error
}

// Reader is an interface for reading messages from a websocket connection
type Reader interface {
	// Read reads a message from the connection
	// Returns the message type, message data, and any error
	Read(conn *websocket.Conn) (messageType websocket.MessageType, data []byte, err error)
}

// ReaderFunc is a function type that implements the Reader interface
type ReaderFunc func(conn *websocket.Conn) (messageType websocket.MessageType, data []byte, err error)

// Read implements the Reader interface for ReaderFunc
func (f ReaderFunc) Read(conn *websocket.Conn) (messageType websocket.MessageType, data []byte, err error) {
	return f(conn)
}

// WriterFunc is a function type that implements the Writer interface
type WriterFunc func(conn *websocket.Conn, messageType websocket.MessageType, data []byte) error

// Write implements the Writer interface for WriterFunc
func (f WriterFunc) Write(conn *websocket.Conn, messageType websocket.MessageType, data []byte) error {
	return f(conn, messageType, data)
}

// NoOpInterceptor implements the Interceptor interface with no-op methods.
// It's used as a fallback when no interceptors are configured or as a base
// struct that other interceptors can embed to avoid implementing all methods.
type NoOpInterceptor struct{}

// BindSocketConnection is a no-op implementation that accepts any connection.
func (interceptor *NoOpInterceptor) BindSocketConnection(_ *websocket.Conn) error {
	return nil
}

// BindSocketWriter returns the original writer without modification.
func (interceptor *NoOpInterceptor) BindSocketWriter(writer Writer) Writer {
	return writer
}

// BindSocketReader returns the original reader without modification.
func (interceptor *NoOpInterceptor) BindSocketReader(reader Reader) Reader {
	return reader
}

func (interceptor *NoOpInterceptor) UnBindSocketConnection(_ *websocket.Conn) {}

// UnBindSocketWriter performs no cleanup operations.
func (interceptor *NoOpInterceptor) UnBindSocketWriter(_ Writer) {}

// UnBindSocketReader performs no cleanup operations.
func (interceptor *NoOpInterceptor) UnBindSocketReader(_ Reader) {}

// Close performs no cleanup operations.
func (interceptor *NoOpInterceptor) Close() error {
	return nil
}
