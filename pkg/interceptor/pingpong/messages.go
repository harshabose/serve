package pingpong

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

var (
	protocolMap = map[message.Protocol]message.Message{
		ProtocolPing: &Ping{},
		ProtocolPong: &Pong{},
	}
)

func ProtocolUnmarshal(protocol message.Protocol, data json.RawMessage) (message.Message, error) {
	msg, exists := protocolMap[protocol]
	if !exists {
		return nil, errors.New("protocol no match")
	}

	if err := msg.Unmarshal(data); err != nil {
		return nil, err
	}

	return msg, nil
}

// Ping represents a connection health check message sent by the server.
// Each ping contains a unique message ID and a timestamp that can be used
// to measure round-trip time when a corresponding pong is received.
type Ping struct {
	message.BaseMessage           // NOTE: EMPTY PAYLOAD
	MessageID           string    `json:"message_id"` // Unique identifier for matching with pong
	Timestamp           time.Time `json:"timestamp"`  // When the ping was sent
}

var ProtocolPing message.Protocol = "ping"

func NewPing(senderID, receiverID string) *Ping {
	return &Ping{
		BaseMessage: message.BaseMessage{
			Header: message.Header{
				SenderID:   senderID,
				ReceiverID: receiverID,
				Protocol:   message.NoneProtocol,
			},
			Payload: nil,
		},
		MessageID: uuid.NewString(),
		Timestamp: time.Now(),
	}
}

// Validate checks if the ping message contains valid data.
// Currently, this is a placeholder for future validation logic.
// Future implementations could validate the message ID format and
// ensure the timestamp is within an acceptable range.
//
// Returns:
//   - An error if validation fails, nil otherwise
func (payload *Ping) Validate() error {
	if payload.MessageID == "" {
		return message.ErrorNotValid
	}
	return payload.BaseMessage.Validate()
}

func (payload *Ping) Protocol() message.Protocol {
	return ProtocolPing
}

// Pong represents a response to a ping message, confirming connection health.
// It contains the original ping's message ID and timestamp, plus its own timestamp,
// allowing the server to calculate the round-trip time.
type Pong struct {
	message.BaseMessage           // NOTE: EMPTY PAYLOAD
	MessageID           string    `json:"message_id"`     // Unique identifier for matching with pong
	Timestamp           time.Time `json:"timestamp"`      // When the ping was sent
	PingTimestamp       time.Time `json:"ping_timestamp"` // When the original ping was sent
}

func NewPong(senderID string, ping *Ping) *Pong {
	return &Pong{
		BaseMessage: message.BaseMessage{
			Header: message.Header{
				SenderID:   senderID,
				ReceiverID: ping.SenderID,
				Protocol:   message.NoneProtocol,
			},
			Payload: nil,
		},
		MessageID:     ping.MessageID,
		Timestamp:     time.Now(),
		PingTimestamp: ping.Timestamp,
	}
}

var ProtocolPong message.Protocol = "pong"

func (payload *Pong) Protocol() message.Protocol {
	return ProtocolPong
}

func (payload *Pong) Validate() error {
	if payload.MessageID == "" {
		return message.ErrorNotValid
	}
	return payload.BaseMessage.Validate()
}
