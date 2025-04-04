package interceptor

import (
	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

// Chain implements the Interceptor interface by combining multiple interceptors
// into a sequential processing pipeline. Each interceptor in the chain gets a chance
// to process the connection, reader, and writer in the order they were added.
// This allows for composing multiple interceptors to build complex behaviors.
type Chain struct {
	interceptors []Interceptor
}

// CreateChain constructs a new Chain that will apply the provided interceptors
// in sequence. The order of interceptors in the slice determines their
// execution order in the chain.
//
// Parameters:
//   - interceptors: Slice of interceptors to include in the chain
//
// Returns:
//   - A new Chain that wraps the provided interceptors
func CreateChain(interceptors []Interceptor) *Chain {
	return &Chain{interceptors: interceptors}
}

// BindSocketConnection binds a WebSocket connection to all interceptors in the chain.
// It passes intercepted writers and readers to each interceptor, ensuring that
// each interceptor receives a fully processed writer and reader stack that includes
// transformations from all other interceptors. This creates a complete interception
// pipeline where each interceptor can use the processed stack for state management.
//
// Parameters:
//   - connection: The WebSocket connection to bind
//   - writer: The base writer to intercept
//   - reader: The base reader to intercept
//
// Returns:
//   - Error if any interceptor fails to bind
func (chain *Chain) BindSocketConnection(connection Connection, writer Writer, reader Reader) error {
	for _, interceptor := range chain.interceptors {
		if err := interceptor.BindSocketConnection(connection, chain.InterceptSocketWriter(writer), chain.InterceptSocketReader(reader)); err != nil {
			return err
		}
	}
	return nil
}

// InterceptSocketWriter applies all writer interceptors in the chain.
// Each interceptor wraps the writer returned by the previous interceptor,
// creating a processing pipeline where the output of one interceptor becomes
// the input to the next.
//
// Parameters:
//   - writer: The base writer to intercept
//
// Returns:
//   - A writer that applies all transformations in the chain
func (chain *Chain) InterceptSocketWriter(writer Writer) Writer {
	for _, interceptor := range chain.interceptors {
		writer = interceptor.InterceptSocketWriter(writer)
	}

	return writer
}

// InterceptSocketReader applies all reader interceptors in the chain.
// Each interceptor wraps the reader returned by the previous interceptor,
// creating a processing pipeline where the output of one interceptor becomes
// the input to the next.
//
// Parameters:
//   - reader: The base reader to intercept
//
// Returns:
//   - A reader that applies all transformations in the chain
func (chain *Chain) InterceptSocketReader(reader Reader) Reader {
	for _, interceptor := range chain.interceptors {
		reader = interceptor.InterceptSocketReader(reader)
	}

	return reader
}

// UnBindSocketConnection notifies all interceptors in the chain that a connection
// is being closed. This allows each interceptor to perform cleanup operations
// for connection-specific resources.
//
// Parameters:
//   - connection: The WebSocket connection to be unbound
func (chain *Chain) UnBindSocketConnection(connection Connection) {
	for _, interceptor := range chain.interceptors {
		interceptor.UnBindSocketConnection(connection)
	}
}

// UnInterceptSocketWriter notifies all interceptors in the chain that a writer
// is being removed. This allows each interceptor to perform cleanup operations
// for writer-specific resources.
//
// Parameters:
//   - writer: The writer being unintercepted
func (chain *Chain) UnInterceptSocketWriter(writer Writer) {
	for _, interceptor := range chain.interceptors {
		interceptor.UnInterceptSocketWriter(writer)
	}
}

// UnInterceptSocketReader notifies all interceptors in the chain that a reader
// is being removed. This allows each interceptor to perform cleanup operations
// for reader-specific resources.
//
// Parameters:
//   - reader: The reader being unintercepted
func (chain *Chain) UnInterceptSocketReader(reader Reader) {
	for _, interceptor := range chain.interceptors {
		interceptor.UnInterceptSocketReader(reader)
	}
}

// Close shuts down all interceptors in the chain and cleans up their resources.
// It collects errors from all interceptors and returns them as a flattened error.
// This method ensures proper cleanup of all resources held by interceptors in the chain.
//
// Returns:
//   - A flattened error containing all errors encountered during shutdown, or nil if successful
func (chain *Chain) Close() error {
	var errs []error
	for _, interceptor := range chain.interceptors {
		errs = append(errs, interceptor.Close())
	}

	return flattenErrs(errs)
}
