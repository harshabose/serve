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
	SetKeys(key, key) error
	SetSessionID(id SessionID)
	Encrypt(string, string, message.Message) (*EncryptedMessage, error)
	Decrypt(*EncryptedMessage) error
	Ready() bool
	Close() error
}

type aes256 struct {
	encryptor cipher.AEAD
	decryptor cipher.AEAD
	sessionID SessionID
	mux       sync.RWMutex
}

func (a *aes256) SetKeys(encKey key, decKey key) error {
	a.mux.Lock()
	defer a.mux.Unlock()

	{
		block, err := aes.NewCipher(encKey[:])
		if err != nil {
			return err
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return err
		}

		a.encryptor = gcm
	}
	{
		block, err := aes.NewCipher(decKey[:])
		if err != nil {
			return err
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return err
		}

		a.decryptor = gcm
	}

	return nil
}

func (a *aes256) SetSessionID(id SessionID) {
	a.mux.Lock()
	defer a.mux.Unlock()

	a.sessionID = id
}

func (a *aes256) Encrypt(senderID, receiverID string, m message.Message) (*EncryptedMessage, error) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	data, err := m.Marshal()
	if err != nil {
		return nil, err
	}

	encryptedData := a.encryptor.Seal(nil, nonce, data, a.sessionID[:])

	encryptedMsg := &EncryptedMessage{
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

func (a *aes256) Decrypt(m *EncryptedMessage) error {
	a.mux.Lock()
	defer a.mux.Unlock()

	data, err := a.decryptor.Open(nil, m.Nonce, m.Payload, a.sessionID[:])
	if err != nil {
		return err
	}

	m.Payload = data

	return nil
}

func (a *aes256) Ready() bool {
	a.mux.RLock()
	defer a.mux.RUnlock()

	if a.encryptor != nil && a.decryptor != nil && !IsZero(a.sessionID) {
		return true
	}

	return false
}

func (a *aes256) Close() error {
	a.mux.Lock()
	defer a.mux.Unlock()

	a.sessionID = SessionID{}
	a.encryptor = nil
	a.decryptor = nil

	return nil
}
