package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"sync"
	"time"

	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type encryptor interface {
	SetKey(key []byte) error
	Encrypt(string, string, message.Message) (*Encrypted, error)
	Decrypt(*Encrypted) error
	Close() error
}

type aes256 struct {
	encryptor cipher.AEAD
	sessionID []byte
	mux       sync.RWMutex
}

func (a *aes256) SetKey(key []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	sessionID := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, sessionID); err != nil {
		return err
	}

	a.encryptor = gcm
	a.sessionID = sessionID

	return nil
}

func (a *aes256) Encrypt(senderID, receiverID string, m message.Message) (*Encrypted, error) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	data, err := m.Marshal()
	if err != nil {
		return nil, err
	}

	encryptedData := a.encryptor.Seal(nil, nonce, data, a.sessionID)

	encryptedMsg := &Encrypted{
		BaseMessage: message.BaseMessage{
			Header: message.Header{
				SenderID:   senderID,
				ReceiverID: receiverID,
				Protocol:   m.Protocol(),
			},
			Payload: encryptedData,
		},
		Nonce:     nonce,
		Timestamp: time.Now(),
	}

	return encryptedMsg, nil
}

func (a *aes256) Decrypt(m *Encrypted) error {
	a.mux.Lock()
	defer a.mux.Unlock()

	data, err := a.encryptor.Open(nil, m.Nonce, m.Payload, a.sessionID)
	if err != nil {
		return err
	}

	m.Payload = data

	return nil
}

func (a *aes256) Close() error {
	a.mux.Lock()
	defer a.mux.Unlock()

	a.sessionID = nil
	a.encryptor = nil

	return nil
}
