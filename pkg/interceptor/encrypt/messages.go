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
)

func CreateMessage(senderID, receiverID string, payload interceptor.Payload) (*interceptor.BaseMessage, error) {
	data, err := payload.Marshal()
	if err != nil {
		return nil, err
	}

	return &interceptor.BaseMessage{
		BaseMessage: message.BaseMessage{
			Header: message.Header{
				SenderID:   senderID,
				ReceiverID: receiverID,
				Protocol:   interceptor.IProtocol,
			},
			Payload: data,
		},
		Header: interceptor.Header{

			MainType: MainType,
			SubType:  payload.Type(),
		},
	}, nil
}

type Encrypted struct {
	Data      []byte    `json:"data"`
	Nonce     []byte    `json:"nonce"`
	Timestamp time.Time `json:"timestamp"`
}

func (payload *Encrypted) Marshal() ([]byte, error) {
	return json.Marshal(payload)
}

func (payload *Encrypted) Unmarshal(data []byte) error {
	return json.Unmarshal(data, payload)
}

func (payload *Encrypted) Validate() error {
	if payload.Data == nil || payload.Nonce == nil || len(payload.Data) <= 0 || len(payload.Nonce) <= 0 {
		return errors.New("not valid")
	}

	return nil
}

func (payload *Encrypted) Process(header interceptor.Header, i interceptor.Interceptor, connection interceptor.Connection) error {
	// TODO implement me
	panic("implement me")
}

func (payload *Encrypted) Type() interceptor.SubType {
	return EncryptedSubType
}
