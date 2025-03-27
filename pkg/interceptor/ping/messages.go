package ping

import (
	"encoding/json"
	"time"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type Message struct {
	interceptor.BaseMessage
}

func CreateMessage(senderID, receiverID string, payload interceptor.Payload) (*Message, error) {
	data, err := payload.Marshal()
	if err != nil {
		return nil, err
	}

	return &Message{
		BaseMessage: interceptor.BaseMessage{
			Header: interceptor.Header{
				SenderID:   senderID,
				ReceiverID: receiverID,
			},
			Payload: data,
		},
	}, nil
}

func (message *Message) Marshal() ([]byte, error) {
	return json.Marshal(message)
}

func (message *Message) Unmarshal(data []byte) error {
	return json.Unmarshal(data, message)
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
