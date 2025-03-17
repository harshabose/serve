package ping

import (
	"encoding/json"
	"time"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

// Payload defines the interface for ping/pong protocol message contents.
// It extends the base message.Message interface with validation and processing
// capabilities specific to the ping/pong protocol. Each implementation represents
// a different message type within the protocol.
//
// Implementations must be able to validate their own content and process
// themselves against the ping/pong state manager when received.
type Payload interface {
	message.Message
	// Validate checks if the payload data is well-formed and valid
	// according to the protocol requirements.
	Validate() error
	// Process handles the payload-specific logic when a message is received,
	// updating the appropriate state in the manager for the given connection.
	Process(message.Header, interceptor.Interceptor, interceptor.Connection) error
}

// Message represents a complete ping/pong protocol message that combines
// routing information (Header) with the actual message content (Payload).
// It encapsulates all information needed to route and process a ping/pong
// message within the system.
type Message struct {
	message.Header         // Contains sender and receiver identification
	Payload        Payload // The actual message content (Ping or Pong)
}

// CreateMessage constructs a new Message with the specified sender, receiver,
// and payload. This factory function ensures messages are properly initialized
// with all required fields.
//
// Parameters:
//   - senderID: Identifier of the sending entity
//   - receiverID: Identifier of the intended receiving entity
//   - msg: The payload content (either Ping or Pong)
//
// Returns:
//   - A fully initialized Message ready for transmission
func CreateMessage(senderID string, receiverID string, msg Payload) *Message {
	return &Message{
		Header: message.Header{
			SenderID:   senderID,
			ReceiverID: receiverID,
		},
		Payload: msg,
	}
}

// Marshal serializes the message into a JSON byte array for transmission.
// This enables the message to be sent over the websocket connection.
//
// Returns:
//   - The JSON-encoded message as a byte array
//   - Any error encountered during serialization
func (msg *Message) Marshal() ([]byte, error) {
	return json.Marshal(msg)
}

// Unmarshal deserializes a JSON byte array into this message structure.
// This processes received data from the websocket connection.
//
// Parameters:
//   - data: The JSON-encoded message as a byte array
//
// Returns:
//   - Any error encountered during deserialization
func (msg *Message) Unmarshal(data []byte) error {
	return json.Unmarshal(data, msg)
}

// Ping represents a connection health check message sent by the server.
// Each ping contains a unique message ID and a timestamp that can be used
// to measure round-trip time when a corresponding pong is received.
type Ping struct {
	MessageID string    `json:"message_id"` // Unique identifier for matching with pong
	Timestamp time.Time `json:"timestamp"`  // When the ping was sent
}

// Marshal serializes the ping payload into a JSON byte array.
// This is typically used when the ping is embedded in a Message.
//
// Returns:
//   - The JSON-encoded ping as a byte array
//   - Any error encountered during serialization
func (payload *Ping) Marshal() ([]byte, error) {
	return json.Marshal(payload)
}

// Unmarshal deserializes a JSON byte array into this ping structure.
// This processes ping data received from a websocket message.
//
// Parameters:
//   - data: The JSON-encoded ping as a byte array
//
// Returns:
//   - Any error encountered during deserialization
func (payload *Ping) Unmarshal(data []byte) error {
	return json.Unmarshal(data, payload)
}

// Validate checks if the ping message contains valid data.
// Currently, this is a placeholder for future validation logic.
// Future implementations could validate the message ID format and
// ensure the timestamp is within an acceptable range.
//
// Returns:
//   - An error if validation fails, nil otherwise
func (payload *Ping) Validate() error {
	return nil
}

// Pong represents a response to a ping message, confirming connection health.
// It contains the original ping's message ID and timestamp, plus its own timestamp,
// allowing the server to calculate the round-trip time.
type Pong struct {
	MessageID     string    `json:"message_id"`     // Matches the corresponding ping's ID
	PingTimestamp time.Time `json:"ping_timestamp"` // When the original ping was sent
	Timestamp     time.Time `json:"timestamp"`      // When this pong was generated
}

// Marshal serializes the pong payload into a JSON byte array.
// This is typically used when the pong is embedded in a Message.
//
// Returns:
//   - The JSON-encoded pong as a byte array
//   - Any error encountered during serialization
func (payload *Pong) Marshal() ([]byte, error) {
	return json.Marshal(payload)
}

// Unmarshal deserializes a JSON byte array into this pong structure.
// This processes pong data received from a websocket message.
//
// Parameters:
//   - data: The JSON-encoded pong as a byte array
//
// Returns:
//   - Any error encountered during deserialization
func (payload *Pong) Unmarshal(data []byte) error {
	return json.Unmarshal(data, payload)
}

// Validate checks if the pong message contains valid data.
// Currently, this is a placeholder for future validation logic.
// Future implementations could validate the message ID format and
// ensure the timestamps are within acceptable ranges.
//
// Returns:
//   - An error if validation fails, nil otherwise
func (payload *Pong) Validate() error {
	return nil
}
