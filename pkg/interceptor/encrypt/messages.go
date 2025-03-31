package encrypt

import (
	"encoding/json"
)

type PayloadType string

const (
	PayloadEncryptedType PayloadType = "encrypt:encrypted"
)

type Message struct {
	Type    PayloadType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func CreateMessage(_type PayloadType, payload json.RawMessage) *Message {
	return &Message{
		Type:    _type,
		Payload: payload,
	}
}
