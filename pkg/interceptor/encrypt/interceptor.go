package encrypt

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/coder/websocket"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/hkdf"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
	"github.com/harshabose/skyline_sonata/serve/pkg/utils"
)

// IsZero is a generic function to check if a value is the zero value for its type
func IsZero[T comparable](value T) bool {
	var zero T
	return value == zero
}

// Crypto-related type definitions for improved type safety
type (
	PrivateKey [32]byte
	PublicKey  [32]byte
	Salt       [16]byte
	SessionID  [16]byte
	Nonce      [12]byte
	key        [32]byte
)

var (
	// ServerPublicKey holds the public key for server verification
	// Loaded from environment variable
	ServerPublicKey []byte
)

// init loads the server keys from environment variables and validates them
func init() {
	ServerPublicKey = []byte(os.Getenv("SERVER_ENCRYPT_PUB_KEY"))
	if len(ServerPublicKey) == 0 {
		fmt.Println("WARNING: SERVER_ENCRYPT_PUB_KEY environment variable not set")
	}
}

// Interceptor implements the encryption interceptor
type Interceptor struct {
	interceptor.NoOpInterceptor
	states          map[interceptor.Connection]*state
	encryptorFactor EncryptorFactory
	isServer        bool
}

func (i *Interceptor) BindSocketConnection(connection interceptor.Connection, writer interceptor.Writer, reader interceptor.Reader) (interceptor.Writer, interceptor.Reader, error) {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	_, exists := i.states[connection]
	if exists {
		return nil, nil, errors.New("connection already exists")
	}

	ctx, cancel := context.WithCancel(i.Ctx)
	encryptor, err := i.encryptorFactor()
	if err != nil {
		cancel()
		return nil, nil, err
	}

	i.states[connection] = &state{
		peerID:    "unknown",
		encryptor: encryptor,
		writer:    writer,
		reader:    reader,
		cancel:    cancel,
		ctx:       ctx,
	}

	return writer, reader, nil
}

func (i *Interceptor) Init(connection interceptor.Connection) error {
	i.Mutex.Lock()

	state, exists := i.states[connection]
	if !exists {
		i.Mutex.Unlock()
		return errors.New("connection not registered")
	}

	// Start the key exchange process
	err := i.initialiseKeyExchange(connection)
	i.Mutex.Unlock() // Unlock before waiting for initialization to avoid deadlock

	if err != nil {
		return fmt.Errorf("encryption initialization failed: %w", err)
	}

	// Wait for the key exchange to complete
	return state.waitUntilInit()
}

func (i *Interceptor) InterceptSocketWriter(writer interceptor.Writer) interceptor.Writer {
	return interceptor.WriterFunc(func(connection interceptor.Connection, messageType websocket.MessageType, m message.Message) error {
		i.Mutex.Lock()
		defer i.Mutex.Unlock()

		state, exists := i.states[connection]
		if !exists {
			return writer.Write(connection, messageType, m)
		}

		// Only encrypt if encryption is ready
		if state.encryptor.Ready() {
			encrypted, err := state.encryptor.Encrypt(m.Message().SenderID, m.Message().ReceiverID, m)
			if err != nil {
				return writer.Write(connection, messageType, m)
			}
			return writer.Write(connection, messageType, encrypted)
		}

		// Pass through unencrypted if encryption not ready // TODO: Is this what I want? Should I pass through
		return writer.Write(connection, messageType, m)
	})
}

func (i *Interceptor) InterceptSocketReader(reader interceptor.Reader) interceptor.Reader {
	return interceptor.ReaderFunc(func(connection interceptor.Connection) (websocket.MessageType, message.Message, error) {
		i.Mutex.Lock()
		defer i.Mutex.Unlock()

		messageType, m, err := reader.Read(connection)
		if err != nil {
			return messageType, m, err
		}

		_, exists := i.states[connection]
		if !exists {
			return messageType, m, nil
		}

		// Process encrypted messages and protocol messages
		payload, err := message.ProtocolUnmarshal(protocolMap, m.Message().Header.Protocol, m.Message().Payload)
		if err != nil {
			return messageType, m, nil
		}

		if err := payload.Process(i, connection); err != nil {
			fmt.Println("error while processing Encryptor m:", err.Error())
		}

		return messageType, payload.Message(), nil
	})
}

func (i *Interceptor) UnBindSocketConnection(connection interceptor.Connection) {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	state, exists := i.states[connection]
	if !exists {
		fmt.Println("connection does not exists")
		return
	}

	// Cancel the context to stop any ongoing operations
	state.cancel()

	// Clean up encryption resources
	if err := state.encryptor.Close(); err != nil {
		fmt.Printf("Error closing encryptor: %v\n", err)
	}
	delete(i.states, connection)
}

func (i *Interceptor) UnInterceptSocketWriter(_ interceptor.Writer) {}

func (i *Interceptor) UnInterceptSocketReader(_ interceptor.Reader) {}

func (i *Interceptor) Close() error {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	var merr = utils.NewMultiError()
	for conn, state := range i.states {
		state.cancel()
		if err := state.encryptor.Close(); err != nil {
			_ = merr.Add(err)
		}
		delete(i.states, conn)
	}

	return merr.ErrorOrNil()
}

func (i *Interceptor) initialiseKeyExchange(connection interceptor.Connection) error {
	var (
		pubKey    PublicKey
		sessionID SessionID
	)

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered")
	}

	// Generate private key
	if _, err := io.ReadFull(rand.Reader, state.privKey[:]); err != nil {
		return err
	}

	// Calculate public key from private key
	curve25519.ScalarBaseMult((*[32]byte)(&pubKey), (*[32]byte)(&state.privKey))

	// Generate random salt for key derivation
	if _, err := io.ReadFull(rand.Reader, state.salt[:]); err != nil {
		return err
	}

	// Load server private key for signing
	serverPrivKey := []byte(os.Getenv("SERVER_ENCRYPT_PRIV_KEY"))
	if len(serverPrivKey) == 0 && i.isServer {
		return errors.New("server private key not available")
	}

	// Generate signature for authentication
	sign := ed25519.Sign(serverPrivKey, append(pubKey[:], state.salt[:]...))

	// Generate session ID
	if _, err := io.ReadFull(rand.Reader, sessionID[:]); err != nil {
		return err
	}
	state.encryptor.SetSessionID(sessionID)

	// Send initialization message
	return state.writer.Write(connection, websocket.MessageText, NewInitMessage(i.ID, state.peerID, pubKey, sign, state.salt, sessionID))
}

// derive generates encryption keys from shared secret
func derive(shared []byte, salt Salt, info string) (key, key, error) {
	hkdfReader := hkdf.New(sha256.New, shared, salt[:], []byte(info))

	key1 := key{}
	if _, err := io.ReadFull(hkdfReader, key1[:]); err != nil {
		return key{}, key{}, err
	}

	key2 := key{}
	if _, err := io.ReadFull(hkdfReader, key2[:]); err != nil {
		return key{}, key{}, err
	}

	return key1, key2, nil
}
