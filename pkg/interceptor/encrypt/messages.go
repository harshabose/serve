package encrypt

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/coder/websocket"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/ed25519"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

var (
	protocolMap = message.ProtocolRegistry{
		ProtocolMessage:       &EncryptedMessage{},
		ProtocolInit:          &Init{},
		ProtocolResponse:      &InitResponse{},
		ProtocolUpdateSession: &UpdateSession{},
	}
)

type EncryptedMessage struct {
	message.BaseMessage
	Nonce     []byte    `json:"nonce"`
	Timestamp time.Time `json:"timestamp"`
}

var ProtocolMessage message.Protocol = "encrypt-message"

func (payload *EncryptedMessage) Validate() error {
	if payload.Nonce == nil || len(payload.Nonce) <= 0 {
		return message.ErrorNotValid
	}

	return payload.BaseMessage.Validate()
}

func (payload *EncryptedMessage) Process(_interceptor interceptor.Interceptor, connection interceptor.Connection) error {
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

func (payload *EncryptedMessage) Protocol() message.Protocol {
	return ProtocolMessage
}

type Init struct {
	message.BaseMessage
	PublicKey PublicKey `json:"public_key"`
	Signature []byte    `json:"signature"`
	SessionID SessionID `json:"session_id"`
	Salt      Salt      `json:"salt"`
}

var ProtocolInit message.Protocol = "encrypt-init"

func NewInitMessage(senderID, receiverID string, pubKey PublicKey, sign []byte, salt Salt, sessionID SessionID) *Init {
	return &Init{
		BaseMessage: message.BaseMessage{
			Header: message.Header{
				SenderID:   senderID,
				ReceiverID: receiverID,
				Protocol:   message.NoneProtocol,
			},
			Payload: nil,
		},
		PublicKey: pubKey,
		Signature: sign,
		SessionID: sessionID,
		Salt:      salt,
	}
}

func (payload *Init) Validate() error {
	if len(payload.Signature) == 0 {
		return message.ErrorNotValid
	}
	return payload.BaseMessage.Validate()
}

func (payload *Init) Protocol() message.Protocol {
	return ProtocolInit
}

func (payload *Init) Process(_interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return errors.New("invalid interceptor")
	}

	signature := append(payload.PublicKey[:], payload.Salt[:]...)
	if ok := ed25519.Verify(ServerPubKey, signature, payload.Signature); !ok {
		return errors.New("signature did not match")
	}

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered")
	}

	var pubKey PublicKey
	if _, err := io.ReadFull(rand.Reader, state.privKey[:]); err != nil {
		return err
	}
	curve25519.ScalarBaseMult((*[32]byte)(&pubKey), (*[32]byte)(&state.privKey))

	state.peerID = payload.SenderID
	state.salt = payload.Salt

	shared, err := curve25519.X25519(state.privKey[:], payload.PublicKey[:])
	if err != nil {
		return fmt.Errorf("failed to compute shared secret: %w", err)
	}

	encKey, decKey, err := derive(shared, state.salt, i.ID)
	if err != nil {
		return err
	}

	if err := state.encryptor.SetKeys(encKey, decKey); err != nil {
		return err
	}
	state.encryptor.SetSessionID(payload.SessionID)

	return state.writer.Write(connection, websocket.MessageText, NewInitResponseMessage(i.ID, state.peerID, pubKey))
}

type InitResponse struct {
	message.BaseMessage
	PublicKey PublicKey `json:"public_key"`
}

var ProtocolResponse message.Protocol = "encrypt-response"

func NewInitResponseMessage(senderID, receiverID string, pub PublicKey) *InitResponse {
	return &InitResponse{
		BaseMessage: message.BaseMessage{
			Header: message.Header{
				SenderID:   senderID,
				ReceiverID: receiverID,
				Protocol:   message.NoneProtocol,
			},
			Payload: nil,
		},
		PublicKey: pub,
	}
}

func (payload *InitResponse) Process(_interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return errors.New("invalid interceptor")
	}

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered")
	}

	shared, err := curve25519.X25519(state.privKey[:], payload.PublicKey[:])
	if err != nil {
		return err
	}

	decKey, encKey, err := derive(shared, state.salt, i.ID)
	if err != nil {
		return err
	}

	return state.encryptor.SetKeys(encKey, decKey)
}

type UpdateSession struct {
	message.BaseMessage
	SessionID   SessionID `json:"session_id"`
	UpdateAtSeq uint64    `json:"update_at_seq"`
}

var ProtocolUpdateSession message.Protocol = "encrypt-update-session"

func (payload *UpdateSession) Process(_interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return errors.New("invalid interceptor")
	}

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered")
	}

	state.encryptor.SetSessionID(payload.SessionID)

	return nil
}
