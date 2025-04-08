package message

import (
	"encoding/json"
	"errors"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type Protocol string

var NoneProtocol Protocol = "none"

type ProtocolRegistry map[Protocol]Message

type Message interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
	Protocol() Protocol
	Message() *BaseMessage
	Validate() error
	Process(interceptor.Interceptor, interceptor.Connection) error
}

type Header struct {
	SenderID   string   `json:"source_id"`
	ReceiverID string   `json:"destination_id"`
	Protocol   Protocol `json:"protocol"`
}

func (header *Header) Validate() error {
	if header.SenderID == "" || header.ReceiverID == "" || header.Protocol == "" {
		return ErrorNotValid
	}

	return nil
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

func (msg *BaseMessage) Protocol() Protocol {
	return NoneProtocol
}

func (msg *BaseMessage) Message() *BaseMessage {
	return msg
}

func (msg *BaseMessage) Validate() error {
	return msg.Header.Validate()
}

func (msg *BaseMessage) Process(interceptor.Interceptor, interceptor.Connection) error {
	return nil
}

func CreateMessage(senderID, receiverID string, payload Message) (*BaseMessage, error) {
	var (
		data     json.RawMessage = nil
		protocol                 = NoneProtocol
		err      error           = nil
	)

	if payload != nil {
		data, err = payload.Marshal()
		if err != nil {
			return nil, err
		}
		protocol = payload.Protocol()
	}

	return &BaseMessage{
		Header: Header{
			SenderID:   senderID,
			ReceiverID: receiverID,
			Protocol:   protocol,
		},
		Payload: data,
	}, nil
}

func CreateMessageFromData(senderID, receiverID string, protocol Protocol, payload json.RawMessage) *BaseMessage {
	return &BaseMessage{
		Header: Header{
			SenderID:   senderID,
			ReceiverID: receiverID,
			Protocol:   protocol,
		},
		Payload: payload,
	}
}

func ProtocolUnmarshal(registry ProtocolRegistry, protocol Protocol, data json.RawMessage) (Message, error) {
	msg, exists := registry[protocol]
	if !exists {
		return nil, errors.New("protocol no match")
	}

	if err := msg.Unmarshal(data); err != nil {
		return nil, err
	}

	return msg, nil
}
