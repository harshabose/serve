package message

import "encoding/json"

type Protocol string

type Message interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

type Header struct {
	SenderID   string   `json:"source_id"`
	ReceiverID string   `json:"destination_id"`
	Protocol   Protocol `json:"protocol"`
}

type BaseMessage struct {
	Header
	Payload json.RawMessage `json:"payload,omitempty"`
}

func (msg *BaseMessage) Marshal() ([]byte, error) {
	return json.Marshal(msg)
}

func (msg *BaseMessage) Unmarshal(data []byte) error {
	return json.Unmarshal(data, msg)
}
