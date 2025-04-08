package encrypt

import (
	"crypto/rand"
	"errors"
	"io"
	"time"

	"github.com/coder/websocket"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/ed25519"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

var (
	protocolMap = map[message.Protocol]message.Message{
		ProtocolEncrypt: &Encrypted{},
	}
)

type Encrypted struct {
	message.BaseMessage
	Nonce     []byte    `json:"nonce"`
	Timestamp time.Time `json:"timestamp"`
}

var ProtocolEncrypt message.Protocol = "encrypt"

func (payload *Encrypted) Validate() error {
	if payload.Nonce == nil || len(payload.Nonce) <= 0 {
		return message.ErrorNotValid
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
	return ProtocolEncrypt
}

type EncryptionInit struct {
	message.BaseMessage
	PublicKey []byte `json:"public_key"`
	Signature []byte `json:"signature"`
	Salt      []byte `json:"salt"`
}

func CreateEncryptionInit(senderID, receiverID string, pubKey, Sig, Salt []byte) *EncryptionInit {
	return &EncryptionInit{
		BaseMessage: message.BaseMessage{
			Header: message.Header{
				SenderID:   senderID,
				ReceiverID: receiverID,
				Protocol:   message.NoneProtocol,
			},
			Payload: nil,
		},
		PublicKey: pubKey,
		Signature: Sig,
		Salt:      Salt,
	}
}

func (payload *EncryptionInit) Validate() error {
	if len(payload.PublicKey) == 0 && len(payload.Signature) == 0 && len(payload.Salt) == 0 {
		return message.ErrorNotValid
	}
	return payload.BaseMessage.Validate()
}

func (payload *EncryptionInit) Process(_interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return errors.New("invalid interceptor")
	}

	signature := append(payload.PublicKey, payload.Salt...)
	if ok := ed25519.Verify(ServerPubKey, signature, payload.Signature); !ok {
		return errors.New("signature did not match")
	}

	var privKey [32]byte
	var pubKey [32]byte

	if _, err := io.ReadFull(rand.Reader, privKey[:]); err != nil {
		return err
	}

	curve25519.ScalarBaseMult(&pubKey, &privKey)

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered")
	}
	state.peerID = payload.SenderID
	state.pubKey = pubKey[:]
	state.privKey = privKey[:]
	state.sessionID = nil // TODO
	state.salt = payload.Salt

	encKey, decKey, err := derive(nil, state.salt, i.ID)
	if err != nil {
		return err
	}

	if err := state.encryptor.SetKey(encKey, decKey, state.sessionID); err != nil {
		return err
	}

	return state.writer.Write(connection, websocket.MessageText, CreateEncryptionResponse(i.ID, state.peerID, pubKey[:]))
}

type EncryptionResponse struct {
	message.BaseMessage
	PublicKey []byte `json:"public_key"`
}

func CreateEncryptionResponse(senderID, receiverID string, pub []byte) *EncryptionResponse {
	return &EncryptionResponse{
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

func (payload *EncryptionResponse) Validate() error {
	if payload.PublicKey == nil || len(payload.PublicKey) == 0 {
		return message.ErrorNotValid
	}

	return payload.BaseMessage.Validate()
}

func (payload *EncryptionResponse) Process(_interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return errors.New("invalid interceptor")
	}

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered")
	}

	shared, err := curve25519.X25519(state.privKey, payload.PublicKey)
	if err != nil {
		return err
	}

	encKey, decKey, err := derive(shared, state.salt, i.ID)
	if err != nil {
		return err
	}

	return state.encryptor.SetKey(encKey, decKey, state.sessionID)
}
