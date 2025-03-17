package ping

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

// Interceptor implements a ping mechanism to maintain websocket connections.
// It periodically sends ping messages and tracks their responses to monitor
// connection health. This enables detection of stale or unresponsive connections
// and provides metrics for connection quality (RTT, success rate).
// The interceptor embeds NoOpInterceptor to inherit default implementations
// and maintains connection state through the manager.
type Interceptor struct {
	interceptor.NoOpInterceptor
	manager  *manager
	interval time.Duration // Time between ping messages
}

// BindSocketConnection initializes tracking for a new websocket connection.
// It creates a new context derived from the interceptor's context, initializes
// connection state in both the interceptor and the manager, and starts a
// background goroutine to periodically send ping messages on this connection.
//
// Parameters:
//   - connection: The websocket connection to monitor
//   - writer: The writer used to send messages on this connection
//   - reader: The reader used to receive messages from this connection
//
// Returns:
//   - An error if the connection is already being tracked or if manager creation fails
func (ping *Interceptor) BindSocketConnection(connection interceptor.Connection, writer interceptor.Writer, reader interceptor.Reader) error {
	ping.Mutex.Lock()
	defer ping.Mutex.Unlock()

	_, exists := ping.State[connection]
	if exists {
		return errors.New("owner already exists")
	}

	ctx, cancel := context.WithCancel(ping.Ctx)

	ping.State[connection] = interceptor.State{
		ClientID: "unknown", // unknown until first pong
		Ctx:      ctx,
		Cancel:   cancel,
		Writer:   writer, // full-stack writer (this is different from the writer in InterceptSocketWriter)
		Reader:   reader,
	}

	if err := ping.manager.manage(connection); err != nil {
		return err
	}

	go ping.loop(ctx, connection)

	return nil
}

// InterceptSocketWriter wraps a writer to process outgoing ping messages.
// This method identifies ping messages in the outgoing stream and processes them
// through the manager to track statistics. It ensures the receiver ID is set
// correctly and updates the state before passing the message to the underlying writer.
//
// Parameters:
//   - writer: The original writer to wrap
//
// Returns:
//   - A wrapped writer that processes ping messages
func (ping *Interceptor) InterceptSocketWriter(writer interceptor.Writer) interceptor.Writer {
	return interceptor.WriterFunc(func(conn interceptor.Connection, messageType websocket.MessageType, message message.Message) error {
		msg, ok := message.(*Message) // if ok, its Ping or Pong message
		if !ok {
			return writer.Write(conn, messageType, message)
		}

		ping.Mutex.Lock()
		defer ping.Mutex.Unlock()

		if state, exists := ping.State[conn]; exists {
			msg.ReceiverID = state.ClientID
			if err := ping.manager.Process(msg, conn); err != nil {
				fmt.Println("error while processing ping pong message: ", err.Error())
			}
		}

		return writer.Write(conn, messageType, message)
	})
}

// InterceptSocketReader wraps a reader to handle pong responses.
// This method parses incoming messages to identify pong responses,
// processes them through the manager to update statistics,
// and passes the original message data through unchanged.
//
// Parameters:
//   - reader: The original reader to wrap
//
// Returns:
//   - A wrapped reader that processes pong messages and updates statistics
func (ping *Interceptor) InterceptSocketReader(reader interceptor.Reader) interceptor.Reader {
	return interceptor.ReaderFunc(func(conn interceptor.Connection) (websocket.MessageType, message.Message, error) {
		messageType, msg, err := reader.Read(conn)
		if err != nil {
			return messageType, msg, err
		}

		pingMsg, ok := msg.(*Message)
		if !ok {
			return messageType, msg, err
		}

		ping.Mutex.Lock()
		defer ping.Mutex.Unlock()

		if _, exists := ping.State[conn]; exists {
			if err := ping.manager.Process(pingMsg, conn); err != nil {
				fmt.Println("error while processing ping pong message: ", err.Error())
			}
		}

		return messageType, msg, nil
	})
}

// UnBindSocketConnection removes a connection from the interceptor's tracking.
// This is called when a connection is closed, ensuring that resources
// associated with the connection are freed to prevent memory leaks.
// It cancels the context for the connection's ping loop, which stops
// the background goroutine, and removes the connection from the state map.
//
// Parameters:
//   - connection: The websocket connection to remove
//   - writer: The writer associated with this connection (unused)
//   - reader: The reader associated with this connection (unused)
func (ping *Interceptor) UnBindSocketConnection(connection interceptor.Connection) {
	ping.Mutex.Lock()
	defer ping.Mutex.Unlock()

	ping.State[connection].Cancel()
	if err := ping.manager.unmanage(connection); err != nil {
		fmt.Println("error while unbinding connection:", err.Error())
	}
	delete(ping.State, connection)
}

// UnInterceptSocketWriter performs cleanup when a writer is being removed.
// Since the writer references are stored by connection, there's no need for
// special cleanup for individual writers. This is primarily implemented
// to satisfy the Interceptor interface.
//
// Parameters:
//   - writer: The writer being removed (unused)
func (ping *Interceptor) UnInterceptSocketWriter(_ interceptor.Writer) {
	// If left unimplemented, NoOpInterceptor's default implementation will be used
	// But, for reference, this method is implemented
}

// UnInterceptSocketReader performs cleanup when a reader is being removed.
// Since the interceptor doesn't maintain reader-specific state, no specific
// cleanup is needed. This is primarily implemented to satisfy the
// Interceptor interface.
//
// Parameters:
//   - reader: The reader being removed (unused)
func (ping *Interceptor) UnInterceptSocketReader(_ interceptor.Reader) {
	// If left unimplemented, NoOpInterceptor's default implementation will be used
	// But, for reference, this method is implemented
}

// Close shuts down the ping interceptor and cleans up all resources.
// It cancels all connection contexts, which stops the background ping loops,
// nullifies reader and writer references to aid garbage collection,
// clears the state map, and triggers cleanup in the manager.
// This method is safe to call multiple times.
//
// Returns:
//   - Any error encountered during cleanup (currently always nil)
func (ping *Interceptor) Close() error {
	ping.Mutex.Lock()
	defer ping.Mutex.Unlock()

	for _, state := range ping.State {
		state.Cancel()
		state.Reader = nil
		state.Writer = nil
	}
	ping.State = make(map[interceptor.Connection]interceptor.State)
	ping.manager.cleanup()

	return nil
}

// loop runs a periodic ping sender for a specific connection.
// This goroutine sends ping messages at the configured interval until
// the context is canceled (typically when the connection is closed).
// For each ping, it generates a unique message ID, creates a ping message
// with the current timestamp, and sends it using the connection's writer.
// The loop ensures each ping is processed by the manager to track statistics.
//
// Parameters:
//   - ctx: Context that controls the lifetime of this loop
//   - conn: The websocket connection to send pings on
func (ping *Interceptor) loop(ctx context.Context, conn interceptor.Connection) {
	ticker := time.NewTicker(ping.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			payload := &Ping{
				MessageID: uuid.NewString(),
				Timestamp: time.Now(),
			}

			if err := payload.Validate(); err != nil {
				fmt.Println("error while sending ping:", err.Error())
				continue
			}

			ping.Mutex.Lock()

			state, exists := ping.State[conn]
			if !exists {
				fmt.Println("state for connection does not exists")
			}

			msg := CreateMessage("server", state.ClientID, payload)

			if err := state.Writer.Write(conn, websocket.MessageText, msg); err != nil {
				fmt.Println("error while sending ping:", err.Error())
				continue
			}

			ping.Mutex.Unlock()
		}
	}
}
