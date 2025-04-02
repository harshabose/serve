package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"sync"
	"time"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type encryptor interface {
	Encrypt(message.Message) (*interceptor.BaseMessage, error)
	Decrypt(*interceptor.BaseMessage) (message.Message, error)
	Close() error
}

type aes256 struct {
	key       []byte
	encryptor cipher.AEAD
	sessionID []byte
	mux       sync.RWMutex
}

func createAES256() (*aes256, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	sessionID := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, sessionID); err != nil {
		return nil, err
	}

	return &aes256{
		key:       key,
		encryptor: gcm,
		sessionID: sessionID,
	}, nil
}

func (a *aes256) Encrypt(senderID, receiverID string, message message.Message) (*interceptor.BaseMessage, error) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	data, err := message.Marshal()
	if err != nil {
		return nil, err
	}

	a.mux.Lock()
	defer a.mux.Unlock()

	encryptedData := a.encryptor.Seal(nil, nonce, data, a.sessionID)
	payload := &Encrypted{Data: encryptedData, Nonce: nonce, Timestamp: time.Now()}

	return CreateMessage(senderID, receiverID, payload)
}

func (a *aes256) Decrypt(message *Encrypted) (message.Message, error) {
	_, err := a.encryptor.Open(nil, message.Nonce, message.Data, a.sessionID)
	if err != nil {
		return nil, err
	}

	// TODO: figure out how to create a message here

	return nil, nil
}

func (a *aes256) Close() error {
	a.mux.Lock()
	defer a.mux.Unlock()

	a.sessionID = nil
	a.encryptor = nil

	return nil
}
