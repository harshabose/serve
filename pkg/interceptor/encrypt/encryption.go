package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

// Common encryption errors
var (
	ErrEncryptionNotReady = errors.New("encryption not ready")
	ErrInvalidKey         = errors.New("invalid encryption key")
	ErrInvalidNonce       = errors.New("invalid nonce")
)

// Encryptor defines the interface for message encryption and decryption
type Encryptor interface {
	// SetKeys configures the encryption and decryption keys
	SetKeys(encryptKey, decryptKey key) error

	// SetSessionID sets the session identifier for this encryption session
	SetSessionID(id SessionID)

	// Encrypt encrypts a message between sender and receiver
	Encrypt(senderID, receiverID string, message message.Message) (*EncryptedMessage, error)

	// Decrypt decrypts an encrypted message in-place
	Decrypt(*EncryptedMessage) error

	// Ready checks if the encryptor is properly initialized and ready to use
	Ready() bool

	// Close releases resources used by the encryptor
	Close() error
}

type EncryptorFactory func() (Encryptor, error)

// AES256 implements the encryptor interface using AES-256-GCM
type AES256 struct {
	encryptor cipher.AEAD
	decryptor cipher.AEAD
	sessionID SessionID
	mux       sync.RWMutex
}

func NewAES256() (Encryptor, error) {
	return &AES256{}, nil
}

// SetKeys configures the encryption and decryption keys
func (a *AES256) SetKeys(encryptKey key, decryptKey key) error {
	a.mux.Lock()
	defer a.mux.Unlock()

	// Setup encryption AEAD
	encBlock, err := aes.NewCipher(encryptKey[:])
	if err != nil {
		return ErrInvalidKey
	}

	encGCM, err := cipher.NewGCM(encBlock)
	if err != nil {
		return ErrInvalidKey
	}

	// Setup decryption AEAD
	decBlock, err := aes.NewCipher(decryptKey[:])
	if err != nil {
		return ErrInvalidKey
	}

	decGCM, err := cipher.NewGCM(decBlock)
	if err != nil {
		return ErrInvalidKey
	}

	a.encryptor = encGCM
	a.decryptor = decGCM

	return nil
}

// SetSessionID sets the session identifier for this encryption session
func (a *AES256) SetSessionID(id SessionID) {
	a.mux.Lock()
	defer a.mux.Unlock()

	a.sessionID = id
}

// Encrypt encrypts a message between sender and receiver
func (a *AES256) Encrypt(senderID, receiverID string, m message.Message) (*EncryptedMessage, error) {
	if !a.Ready() {
		return nil, ErrEncryptionNotReady
	}

	// Generate random nonce
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Marshal the original message
	data, err := m.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	// Lock only for encryption operation
	a.mux.RLock()
	encryptedData := a.encryptor.Seal(nil, nonce, data, a.sessionID[:])
	a.mux.RUnlock()

	// Create encrypted message wrapper
	encryptedMsg := &EncryptedMessage{
		BaseMessage: message.BaseMessage{
			Header: message.Header{
				SenderID:   senderID,
				ReceiverID: receiverID,
				Protocol:   ProtocolMessage,
			},
			Payload: encryptedData,
		},
		Nonce:     nonce,
		Timestamp: time.Now(),
	}

	return encryptedMsg, nil
}

// Decrypt decrypts an encrypted message in-place
func (a *AES256) Decrypt(m *EncryptedMessage) error {
	if !a.Ready() {
		return ErrEncryptionNotReady
	}

	if m.Nonce == nil || len(m.Nonce) == 0 {
		return ErrInvalidNonce
	}

	a.mux.RLock()
	defer a.mux.RUnlock()

	// Decrypt the payload
	data, err := a.decryptor.Open(nil, m.Nonce, m.Payload, a.sessionID[:])
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	// Replace encrypted payload with decrypted payload
	m.Payload = data

	return nil
}

// Ready checks if the encryptor is properly initialized and ready to use
func (a *AES256) Ready() bool {
	a.mux.RLock()
	defer a.mux.RUnlock()

	return a.encryptor != nil && a.decryptor != nil && !IsZero(a.sessionID)
}

// Close releases resources used by the encryptor
func (a *AES256) Close() error {
	a.mux.Lock()
	defer a.mux.Unlock()

	a.sessionID = SessionID{}
	a.encryptor = nil
	a.decryptor = nil

	return nil
}
