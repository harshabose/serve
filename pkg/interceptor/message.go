package interceptor

import "encoding/json"

type Message interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

type Header struct {
	SenderID   string `json:"source_id"`
	ReceiverID string `json:"destination_id"`
	MainType   string `json:"main_type"`
	SubType    string `json:"sub_type"`
}

type BaseMessage struct {
	Header
	Payload json.RawMessage `json:"payload"`
}
