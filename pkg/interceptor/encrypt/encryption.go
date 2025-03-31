package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"sync"

	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type encryptor interface {
	Encrypt(message.Message) (*Message, error)
	Decrypt(*Message) (message.Message, error)
	Close() error
}

type aes256 struct {
	encryptor cipher.AEAD
	sessionID []byte
	mux       sync.RWMutex
}

func createAES256() (*aes256, error) {
	// generate key
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}

	// create cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// create GCM mode encryptor
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	sessionID := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, sessionID); err != nil {
		return nil, err
	}

	return &aes256{
		encryptor: gcm,
		sessionID: sessionID,
	}, nil
}

func (a *aes256) Encrypt(message message.Message) (*Message, error) {
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
	finalData := make([]byte, 2+len(nonce)+len(encryptedData))
	binary.BigEndian.PutUint16(finalData[:2], uint16(len(nonce)))

	copy(finalData[2:], nonce)
	copy(finalData[2+len(nonce):], encryptedData)

	return CreateMessage(PayloadEncryptedType, finalData), nil
}

func (a *aes256) Decrypt(message *Message) (message.Message, error) {

}

func (a *aes256) Close() error {

}
