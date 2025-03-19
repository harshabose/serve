package interceptor

import (
	"context"
	"io"
	"sync"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/message"
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

// Connection defines a interface
type Connection interface {
	Write(context.Context, []byte) error
	Read(ctx context.Context) ([]byte, error)
}

// Interceptor defines a transformer that can modify the behavior of websocket connections.
// Interceptors can bind to the connection itself, its writers (for outgoing messages),
// and its readers (for incoming messages). This pattern enables adding functionalities
// like logging, encryption, compression, rate limiting, or analytics to websocket
// connections without modifying the core websocket handling code.
type Interceptor interface {
	// BindSocketConnection is called when a new websocket connection is established.
	// It gives the interceptor an opportunity to set up any connection-specific
	// state or perform initialization tasks for the given connection, writer, and reader.
	// Returns an error if the binding process fails, which typically would
	// result in the connection being rejected.
	BindSocketConnection(Connection, Writer, Reader) error

	// InterceptSocketWriter wraps a writer that handles messages going out to clients.
	// The interceptor receives the original writer and returns a modified writer
	// that adds the interceptor's functionality. For example, an encryption interceptor
	// would return a writer that encrypts messages before passing them to the original writer.
	// The returned writer will be used for all future write operations on the connection.
	InterceptSocketWriter(Writer) Writer

	// InterceptSocketReader wraps a reader that handles messages coming in from clients.
	// The interceptor receives the original reader and returns a modified reader
	// that adds the interceptor's functionality. For example, a logging interceptor
	// would return a reader that logs messages after receiving them from the original reader.
	// The returned reader will be used for all future read operations on the connection.
	InterceptSocketReader(Reader) Reader

	// UnBindSocketConnection is called when a websocket connection is closed or removed.
	// It cleans up any connection-specific resources and state maintained by the interceptor
	// for the given connection, as well as associated writer and reader.
	// This prevents memory leaks and ensures proper resource cleanup.
	UnBindSocketConnection(Connection)

	// UnInterceptSocketWriter is called when a writer is being removed or when the
	// connection is closing. This gives the interceptor an opportunity to clean up
	// any resources or state associated with the writer. The interceptor should
	// release any references to the writer to prevent memory leaks.
	UnInterceptSocketWriter(Writer)

	// UnInterceptSocketReader is called when a reader is being removed or when the
	// connection is closing. This gives the interceptor an opportunity to clean up
	// any resources or state associated with the reader. The interceptor should
	// release any references to the reader to prevent memory leaks.
	UnInterceptSocketReader(Reader)

	// Closer interface implementation for resource cleanup.
	// Close is called when the interceptor itself is being shut down.
	// It should clean up any global resources held by the interceptor.
	io.Closer
}

// Writer is an interface for writing messages to a websocket connection
type Writer interface {
	// Write sends a message to the connection
	// Takes the connection, message type, and message to write
	// Returns any error encountered during writing
	Write(conn Connection, messageType websocket.MessageType, message message.Message) error
}

// Reader is an interface for reading messages from a websocket connection
type Reader interface {
	// Read reads a message from the connection
	// Returns the message type, message data, and any error
	Read(conn Connection) (messageType websocket.MessageType, message message.Message, err error)
}

// ReaderFunc is a function type that implements the Reader interface
type ReaderFunc func(conn Connection) (messageType websocket.MessageType, message message.Message, err error)

// Read implements the Reader interface for ReaderFunc
func (f ReaderFunc) Read(conn Connection) (messageType websocket.MessageType, message message.Message, err error) {
	return f(conn)
}

// WriterFunc is a function type that implements the Writer interface
type WriterFunc func(conn Connection, messageType websocket.MessageType, message message.Message) error

// Write implements the Writer interface for WriterFunc
func (f WriterFunc) Write(conn Connection, messageType websocket.MessageType, message message.Message) error {
	return f(conn, messageType, message)
}

type WriterReader struct {
	Writer
	Reader
}

// State holds all the connection-specific state for an interceptor.
// It maintains the context for cancellation, client identification,
// and references to the writer and reader for sending/receiving messages.

// NoOpInterceptor implements the Interceptor interface with no-op methods.
// It's used as a fallback when no interceptors are configured or as a base
// struct that other interceptors can embed to avoid implementing all methods.
// It provides state management for connections with synchronization.
type NoOpInterceptor struct {
	ID    string          // Identifier for this interceptor
	Mutex sync.RWMutex    // Mutex for thread-safe access to State
	Ctx   context.Context // Parent context for all connections
}

// BindSocketConnection is a no-op implementation that accepts any connection.
// Along with the connection, it also receives the writer and reader that will
// be used with this connection, though the base implementation doesn't use them.
func (interceptor *NoOpInterceptor) BindSocketConnection(_ Connection, _ Writer, _ Reader) error {
	return nil
}

// InterceptSocketWriter returns the original writer without modification.
// Derived interceptors would override this to add functionality to the writer.
func (interceptor *NoOpInterceptor) InterceptSocketWriter(writer Writer) Writer {
	return writer
}

// InterceptSocketReader returns the original reader without modification.
// Derived interceptors would override this to add functionality to the reader.
func (interceptor *NoOpInterceptor) InterceptSocketReader(reader Reader) Reader {
	return reader
}

// UnBindSocketConnection performs no cleanup operations in the base implementation.
// It receives the connection being closed along with its writer and reader.
func (interceptor *NoOpInterceptor) UnBindSocketConnection(_ Connection) {}

// UnInterceptSocketWriter performs no cleanup operations in the base implementation.
// Derived classes would override this to clean up resources associated with the writer.
func (interceptor *NoOpInterceptor) UnInterceptSocketWriter(_ Writer) {}

// UnInterceptSocketReader performs no cleanup operations in the base implementation.
// Derived classes would override this to clean up resources associated with the reader.
func (interceptor *NoOpInterceptor) UnInterceptSocketReader(_ Reader) {}

// Close performs no cleanup operations in the base implementation.
// Derived classes would override this to clean up global resources.
func (interceptor *NoOpInterceptor) Close() error {
	return nil
}

// Payload defines the interface for protocol message contents.
// It extends the base message.Message interface with validation and processing
// capabilities specific to the protocol. Each implementation represents
// a different message type within the protocol.
//
// Implementations must be able to validate their own content and process
// themselves against their respective Interceptor when received.
type Payload interface {
	message.Message
	// Validate checks if the payload data is well-formed and valid
	// according to the protocol requirements.
	Validate() error
	// Process handles the payload-specific logic when a message is received,
	// updating the appropriate state in the manager for the given connection.
	Process(message.Header, Interceptor, Connection) error
}
