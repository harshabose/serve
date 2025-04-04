package encrypt

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

var (
	MainType interceptor.MainType = "encrypt"

	EncryptedSubType interceptor.SubType = "encrypted"

	subTypeMap = map[interceptor.SubType]interceptor.Payload{
		EncryptedSubType: &Encrypted{},
	}
)

func PayloadUnmarshal(sub interceptor.SubType, p json.RawMessage) (interceptor.Payload, error) {
	if payload, exists := subTypeMap[sub]; exists {
		if err := payload.Unmarshal(p); err != nil {
			return nil, err
		}
		return payload, nil
	}

	return nil, errors.New("processor does not exist for given type")
}

type Encrypted struct {
	message.BaseMessage
	Data      []byte    `json:"data"`
	Nonce     []byte    `json:"nonce"`
	Timestamp time.Time `json:"timestamp"`
}

var Protocol message.Protocol = "encrypt"

func NewEncrypt(senderID, receiverID string, data, nonce []byte) *Encrypted {
	return &Encrypted{
		BaseMessage: message.BaseMessage{
			Header: message.Header{
				SenderID:   senderID,
				ReceiverID: receiverID,
				Protocol:   message.NoneProtocol,
			},
			Payload: nil,
		},
		Data:      data,
		Nonce:     nonce,
		Timestamp: time.Now(),
	}
}

func (payload *Encrypted) Validate() error {
	if payload.Data == nil || payload.Nonce == nil || len(payload.Data) <= 0 || len(payload.Nonce) <= 0 {
		return errors.New("not valid")
	}

	return payload.BaseMessage.Validate()
}

func (payload *Encrypted) Process(_interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return errors.New("inappropriate interceptor for the payload")
	}

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered")
	}

	msg, err := state.encryptor.Decrypt(payload)
	if err != nil {
		return err
	}

}

func (payload *Encrypted) Protocol() message.Protocol {
	return Protocol
}
