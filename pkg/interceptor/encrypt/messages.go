package encrypt

import (
	"errors"
	"time"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type Encrypted struct {
	message.BaseMessage
	Nonce     []byte    `json:"nonce"`
	Timestamp time.Time `json:"timestamp"`
}

var Protocol message.Protocol = "encrypt"

func (payload *Encrypted) Validate() error {
	if payload.Nonce == nil || len(payload.Nonce) <= 0 {
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
	if err := state.encryptor.Decrypt(payload); err != nil {
		return err
	}

	return nil
}

func (payload *Encrypted) Protocol() message.Protocol {
	return Protocol
}
