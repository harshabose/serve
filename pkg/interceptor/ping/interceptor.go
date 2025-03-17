package ping

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coder/websocket"

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
	internalState map[interceptor.Connection]*state
	interval      time.Duration // Time between ping messages
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
func (i *Interceptor) BindSocketConnection(connection interceptor.Connection, writer interceptor.Writer, reader interceptor.Reader) error {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	_, exists := i.State[connection]
	if exists {
		return errors.New("owner already exists")
	}

	ctx, cancel := context.WithCancel(i.Ctx)

	i.State[connection] = interceptor.State{
		ID:     "unknown", // unknown until first pong
		Ctx:    ctx,
		Cancel: cancel,
		Writer: writer, // full-stack writer (this is different from the writer in InterceptSocketWriter)
		Reader: reader,
	}

	if _, exists := i.internalState[connection]; exists {
		return errors.New("connection already exists")
	}

	i.internalState[connection] = &state{
		pings: make([]*ping, 0),
		pongs: make([]*pong, 0),
		max:   100,
	}

	// SEND INITIAL PING

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
func (i *Interceptor) InterceptSocketWriter(writer interceptor.Writer) interceptor.Writer {
	return interceptor.WriterFunc(func(conn interceptor.Connection, messageType websocket.MessageType, message message.Message) error {
		msg, ok := message.(*Message) // if ok, its Ping or Pong message
		if !ok {
			return writer.Write(conn, messageType, message)
		}

		i.Mutex.Lock()
		defer i.Mutex.Unlock()

		if _, exists := i.State[conn]; exists {
			if err := msg.Payload.Process(msg.Header, i, conn); err != nil {
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
func (i *Interceptor) InterceptSocketReader(reader interceptor.Reader) interceptor.Reader {
	return interceptor.ReaderFunc(func(conn interceptor.Connection) (websocket.MessageType, message.Message, error) {
		messageType, msg, err := reader.Read(conn)
		if err != nil {
			return messageType, msg, err
		}

		Msg, ok := msg.(*Message)
		if !ok {
			return messageType, msg, err
		}

		i.Mutex.Lock()
		defer i.Mutex.Unlock()

		if _, exists := i.State[conn]; exists {
			if err := Msg.Payload.Process(Msg.Header, i, conn); err != nil {
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
func (i *Interceptor) UnBindSocketConnection(connection interceptor.Connection) {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	i.State[connection].Cancel()
	// if err := i.manager.unmanage(connection); err != nil {
	// 	fmt.Println("error while unbinding connection:", err.Error())
	// }
	delete(i.State, connection)
}

// UnInterceptSocketWriter performs cleanup when a writer is being removed.
// Since the writer references are stored by connection, there's no need for
// special cleanup for individual writers. This is primarily implemented
// to satisfy the Interceptor interface.
//
// Parameters:
//   - writer: The writer being removed (unused)
func (i *Interceptor) UnInterceptSocketWriter(_ interceptor.Writer) {
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
func (i *Interceptor) UnInterceptSocketReader(_ interceptor.Reader) {
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
func (i *Interceptor) Close() error {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	for _, state := range i.State {
		state.Cancel()
		state.Reader = nil
		state.Writer = nil
	}
	i.State = make(map[interceptor.Connection]interceptor.State)

	return nil
}

func (payload *Ping) Process(header message.Header, interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	if err := payload.Validate(); err != nil {
		return err
	}

	i := interceptor.(*Interceptor)

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	state, exists := i.State[connection]
	if !exists {
		return errors.New("connection does not exists")
	}

	internalState, exists := i.internalState[connection]
	if !exists {
		return errors.New("connection does not exists")
	}

	internalState.recordPing(payload)

	pong := CreateMessage(i.ID, header.SenderID, &Pong{payload.MessageID, payload.Timestamp, time.Now()})

	return state.Writer.Write(connection, websocket.MessageText, pong)
}

func (payload *Pong) Process(header message.Header, interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	if err := payload.Validate(); err != nil {
		return err
	}

	i := interceptor.(*Interceptor)

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	state, exists := i.State[connection]
	if !exists {
		return errors.New("connection does not exists")
	}

	internalState, exists := i.internalState[connection]
	if !exists {
		return errors.New("connection does not exists")
	}

	internalState.recordPong(payload)

	ping := CreateMessage(i.ID, header.SenderID, &Ping{payload.MessageID, time.Now()})

	return state.Writer.Write(connection, websocket.MessageText, ping)
}
