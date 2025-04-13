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

// Protocol message type constants
var (
	// Protocol identifiers
	ProtocolMessage       message.Protocol = "encrypt-message"
	ProtocolInit          message.Protocol = "encrypt-init"
	ProtocolResponse      message.Protocol = "encrypt-response"
	ProtocolInitDone      message.Protocol = "encrypt-done"
	ProtocolUpdateSession message.Protocol = "encrypt-update-session"

	// Error constants
	ErrInvalidInterceptor   = errors.New("inappropriate interceptor for the payload")
	ErrConnectionNotFound   = errors.New("connection not registered")
	ErrInvalidSignature     = errors.New("signature verification failed")
	ErrInvalidServerRequest = errors.New("invalid request to server")

	// Protocol registry maps protocol identifiers to message types
	protocolMap = message.ProtocolRegistry{
		ProtocolMessage:       &EncryptedMessage{},
		ProtocolInit:          &Init{},
		ProtocolResponse:      &InitResponse{},
		ProtocolInitDone:      &InitDone{},
		ProtocolUpdateSession: &UpdateSession{},
	}
)

// EncryptedMessage represents an encrypted payload
type EncryptedMessage struct {
	message.BaseMessage
	Nonce     []byte    `json:"nonce"`
	Timestamp time.Time `json:"timestamp"`
}

// Validate ensures the encrypted message contains valid data
func (payload *EncryptedMessage) Validate() error {
	if payload.Nonce == nil || len(payload.Nonce) <= 0 {
		return message.ErrorNotValid
	}

	return payload.BaseMessage.Validate()
}

// Process handles decryption of the message
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

// Protocol returns the message protocol type
func (payload *EncryptedMessage) Protocol() message.Protocol {
	return ProtocolMessage
}

// Init represents the initial key exchange message
type Init struct {
	message.BaseMessage
	PublicKey PublicKey `json:"public_key"`
	Signature []byte    `json:"signature"`
	SessionID SessionID `json:"session_id"`
	Salt      Salt      `json:"salt"`
}

// NewInitMessage creates a new initialization message for key exchange
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

// Validate checks if the init message contains valid data
func (payload *Init) Validate() error {
	if len(payload.Signature) == 0 {
		return message.ErrorNotValid
	}
	return payload.BaseMessage.Validate()
}

// Protocol returns the message protocol type
func (payload *Init) Protocol() message.Protocol {
	return ProtocolInit
}

// Process handles the initialization message
func (payload *Init) Process(_interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return ErrInvalidInterceptor
	}

	// Verify signature using server public key
	signature := append(payload.PublicKey[:], payload.Salt[:]...)
	if !ed25519.Verify(ServerPublicKey, signature, payload.Signature) {
		return ErrInvalidSignature
	}

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered")
	}

	// Generate key pair for this connection
	var pubKey PublicKey
	if _, err := io.ReadFull(rand.Reader, state.privKey[:]); err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}
	curve25519.ScalarBaseMult((*[32]byte)(&pubKey), (*[32]byte)(&state.privKey))

	// Save peer information
	state.peerID = payload.SenderID
	state.salt = payload.Salt

	// Compute shared secret and derive keys
	shared, err := curve25519.X25519(state.privKey[:], payload.PublicKey[:])
	if err != nil {
		return fmt.Errorf("failed to compute shared secret: %w", err)
	}

	// Derive encryption and decryption keys
	encKey, decKey, err := derive(shared, state.salt, i.ID)
	if err != nil {
		return fmt.Errorf("key derivation failed: %w", err)
	}

	// Configure encryptor with derived keys
	if err := state.encryptor.SetKeys(encKey, decKey); err != nil {
		return err
	}
	state.encryptor.SetSessionID(payload.SessionID)

	// Send response with the public key
	return state.writer.Write(connection, websocket.MessageText, NewInitResponseMessage(i.ID, state.peerID, pubKey))
}

// InitResponse represents the response to an initialization message
type InitResponse struct {
	message.BaseMessage
	PublicKey PublicKey `json:"public_key"`
	// NOTE: NO SIGNING HERE. AUTH IS DONE SEPARATELY
}

// NewInitResponseMessage creates a new response message for key exchange
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

func (payload *InitResponse) Protocol() message.Protocol {
	return ProtocolResponse
}

// Process handles the initialization response
func (payload *InitResponse) Process(_interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return errors.New("invalid interceptor")
	}

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered")
	}

	// Save peer ID for future communications
	state.peerID = payload.SenderID

	// Compute shared secret using our private key and peer's public key
	shared, err := curve25519.X25519(state.privKey[:], payload.PublicKey[:])
	if err != nil {
		return err
	}

	// For responses, keys are reversed compared to the initiation
	decKey, encKey, err := derive(shared, state.salt, i.ID) // NOTE: KEY REVERSED
	if err != nil {
		return err
	}

	// Configure encryptor with derived keys
	if err := state.encryptor.SetKeys(encKey, decKey); err != nil {
		return err
	}

	// Signal that initialization is complete
	select {
	case state.initDone <- struct{}{}:
	default:
		// Channel already has a value, which is fine
	}

	// Send acknowledgment
	return state.writer.Write(connection, websocket.MessageText, NewInitDoneMessage(i.ID, state.peerID))
}

// InitDone represents the acknowledgment that key exchange is complete
type InitDone struct {
	message.BaseMessage
}

// NewInitDoneMessage creates a new completion message for key exchange
func NewInitDoneMessage(senderID, receiverID string) *InitDone {
	return &InitDone{
		BaseMessage: message.BaseMessage{
			Header: message.Header{
				SenderID:   senderID,
				ReceiverID: receiverID,
				Protocol:   message.NoneProtocol,
			},
			Payload: nil,
		},
	}
}

// Protocol returns the message protocol type
func (payload *InitDone) Protocol() message.Protocol {
	return ProtocolInitDone
}

// Process handles the initialization completion message
func (payload *InitDone) Process(_interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return errors.New("invalid interceptor")
	}

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered")
	}

	// Signal that initialization is complete
	select {
	case state.initDone <- struct{}{}:
	default:
		// Channel already has a value, which is fine
	}

	state.initDone <- struct{}{}

	return nil
}

// UpdateSession represents a message to update the session ID
type UpdateSession struct {
	message.BaseMessage
	SessionID   SessionID `json:"session_id"`
	UpdateAtSeq uint64    `json:"update_at_seq"`
}

// Process handles session update requests
func (payload *UpdateSession) Process(_interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return errors.New("invalid interceptor")
	}

	// Only clients should process update session messages
	if i.isServer {
		return ErrInvalidServerRequest
	}

	state, exists := i.states[connection]
	if !exists {
		return ErrConnectionNotFound
	}

	// Update the session ID
	state.encryptor.SetSessionID(payload.SessionID)

	return nil
}
