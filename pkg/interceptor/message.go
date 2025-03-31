package interceptor

import "encoding/json"

type Message interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

type (
	Protocol string
	SubType  string
	MainType string
)

type Header struct {
	SenderID   string   `json:"source_id"`
	ReceiverID string   `json:"destination_id"`
	Protocol   Protocol `json:"protocol"`
	MainType   MainType `json:"main_type"`
	SubType    SubType  `json:"sub_type"`
}

var IProtocol Protocol = "interceptor"

type BaseMessage struct { // This actually needs to be interceptor module and Message interface should be in its own module
	Header
	Payload json.RawMessage `json:"payload"`
}

func (msg *BaseMessage) Marshal() ([]byte, error) {
	return json.Marshal(msg)
}

func (msg *BaseMessage) Unmarshal(data []byte) error {
	return json.Unmarshal(data, msg)
}
