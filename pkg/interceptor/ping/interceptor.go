package ping

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

// collection combines connection-specific statistics and a writer reference,
// storing all per-connection state needed by the ping interceptor.
type collection struct {
	*pings
	interceptor.Writer
}

// Interceptor implements a ping mechanism to maintain websocket connections.
// It periodically sends ping messages and tracks their responses to monitor
// connection health.
type Interceptor struct {
	interceptor.NoOpInterceptor
	interval     time.Duration
	statsFactory statsFactory
	collection   map[*websocket.Conn]*collection
	mux          sync.RWMutex
	close        chan struct{}
	ctx          context.Context
}

// BindSocketConnection initializes tracking for a new websocket connection by
// creating pings for it and storing it in the collection map. The writer will
// be set later when BindSocketWriter is called for this connection.
func (ping *Interceptor) BindSocketConnection(connection *websocket.Conn) error {
	ping.mux.Lock()
	defer ping.mux.Unlock()

	stats, err := ping.statsFactory.createStats()
	if err != nil {
		return err
	}

	ping.collection[connection] = &collection{stats, nil}
	return nil
}

// BindSocketWriter wraps a writer to store the writer for later.
func (ping *Interceptor) BindSocketWriter(writer interceptor.Writer) interceptor.Writer {
	return interceptor.WriterFunc(func(conn *websocket.Conn, messageType websocket.MessageType, data []byte) error {
		// Store the writer for this connection
		// Storing the writer allows other interceptors to perform their interceptions as well
		ping.mux.Lock()
		defer ping.mux.Unlock()
		if _, exists := ping.collection[conn]; exists {
			ping.collection[conn].Writer = writer
		}

		// Pass through to original writer
		// No manipulation of writer, just storing
		return writer.Write(conn, messageType, data)
	})
}

// BindSocketReader wraps a reader to handle pong responses.
func (ping *Interceptor) BindSocketReader(reader interceptor.Reader) interceptor.Reader {
	return interceptor.ReaderFunc(func(conn *websocket.Conn) (websocket.MessageType, []byte, error) {
		messageType, data, err := reader.Read(conn)
		if err != nil {
			return messageType, data, err
		}

		msg := &message.Pong{}
		if err := msg.Unmarshal(data); err == nil {
			// Message is Pong message
			ping.collection[conn].pings.recordPong(msg)
		}

		return messageType, data, nil
	})
}

// UnBindSocketConnection removes a connection from the interceptor's tracking.
// This is called when a connection is closed, ensuring that resources
// associated with the connection are freed and preventing memory leaks.
func (ping *Interceptor) UnBindSocketConnection(connection *websocket.Conn) {
	ping.mux.Lock()
	defer ping.mux.Unlock()

	delete(ping.collection, connection)
}

// UnBindSocketWriter performs cleanup when a writer is being removed.
// Since the writer references are stored by connection, writer don't need
// special cleanup for individual writers.
func (ping *Interceptor) UnBindSocketWriter(_ interceptor.Writer) {
	// If left, unimplemented, NoOpInterceptor's default implementation will be used
	// But, for reference, this method is implemented
}

// UnBindSocketReader performs cleanup when a reader is being removed.
// Since the Interceptor don't maintain reader-specific state, no specific cleanup is needed.
func (ping *Interceptor) UnBindSocketReader(_ interceptor.Reader) {
	// If left, unimplemented, NoOpInterceptor's default implementation will be used
	// But, for reference, this method is implemented
}

// Close shuts down the ping interceptor and cleans up all resources.
// It signals the background ping loop to stop, waits for confirmation
// that it has stopped, and cleans up any remaining connection state.
// This method is safe to call multiple times.
func (ping *Interceptor) Close() error {
	select {
	case ping.close <- struct{}{}:
		// sent signal successfully
	default:
		// already closing/closed
	}

	ping.mux.Lock()
	defer ping.mux.Unlock()
	ping.collection = make(map[*websocket.Conn]*collection)

	return nil
}

func (ping *Interceptor) loop() {
	ticker := time.NewTicker(ping.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ping.ctx.Done():
			return
		case <-ping.close:
			return
		case <-ticker.C:
			ping.mux.RLock()

			// Send ping messages to all connections
			for conn, collection := range ping.collection {
				if collection.Writer == nil {
					fmt.Println("writer not bound yet; skipping...")
					continue
				}

				msg := message.CreatePingMessage(time.Now())
				data, err := msg.Marshal()
				if err != nil {
					fmt.Println("error while marshaling ping message; skipping...")
					continue
				}

				// Use the stored writer instead of sending through websocket.Conn.Write(...)
				if err := collection.Writer.Write(conn, websocket.MessageText, data); err != nil {
					fmt.Println("error while sending ping message; skipping...")
					continue
				}

				// Record successful ping
				collection.pings.recordSentPing(msg)
			}

			ping.mux.RUnlock()
		}
	}
}

func (ping *Interceptor) sendPing() {
	ping.mux.RLock()
	defer ping.mux.RUnlock()

}
