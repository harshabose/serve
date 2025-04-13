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
)

func IsZero[T comparable](value T) bool {
	var zero T
	return value == zero
}

type (
	PrivateKey [32]byte
	PublicKey  [32]byte
	Salt       [16]byte
	SessionID  [16]byte
	Nonce      [12]byte
	key        [32]byte
)

var SeverPublicKey = []byte(os.Getenv("SERVER_ENCRYPT_PUB_KEY"))

type Interceptor struct {
	interceptor.NoOpInterceptor
	states    map[interceptor.Connection]*state
	iamserver bool
}

func (i *Interceptor) BindSocketConnection(connection interceptor.Connection, writer interceptor.Writer, reader interceptor.Reader) (interceptor.Writer, interceptor.Reader, error) {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	_, exists := i.states[connection]
	if exists {
		return nil, nil, errors.New("connection already exists")
	}

	ctx, cancel := context.WithCancel(i.Ctx)
	i.states[connection] = &state{
		peerID:    "unknown",
		encryptor: &aes256{},
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
		return errors.New("connection not registered")
	}

	if err := i.init(connection); err != nil {
		return err
	}
	i.Mutex.Unlock()

	return state.waitUntilInit()
}

func (i *Interceptor) InterceptSocketWriter(writer interceptor.Writer) interceptor.Writer {
	/*
		Takes in any type of message.Message and encrypts it. In general, all implementations of
		message.Message should use message.BaseMessage to implement message.Message.
	*/
	return interceptor.WriterFunc(func(connection interceptor.Connection, messageType websocket.MessageType, m message.Message) error {
		i.Mutex.Lock()
		defer i.Mutex.Unlock()

		state, exists := i.states[connection]
		if !exists {
			return writer.Write(connection, messageType, m)
		}

		if state.encryptor.Ready() {
			encrypted, err := state.encryptor.Encrypt(m.Message().SenderID, m.Message().ReceiverID, m)
			if err != nil {
				return writer.Write(connection, messageType, m)
			}
			return writer.Write(connection, messageType, encrypted)
		}

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

		payload, err := message.ProtocolUnmarshal(protocolMap, m.Message().Header.Protocol, m.Message().Payload)
		if err != nil {
			return messageType, m, nil
		}

		if err := payload.Process(i, connection); err != nil {
			fmt.Println("error while processing encryptor m:", err.Error())
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

	state.cancel()
	delete(i.states, connection)
}

func (i *Interceptor) UnInterceptSocketWriter(_ interceptor.Writer) {}

func (i *Interceptor) UnInterceptSocketReader(_ interceptor.Reader) {}

func (i *Interceptor) Close() error {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	i.states = make(map[interceptor.Connection]*state)

	return nil
}

func (i *Interceptor) init(connection interceptor.Connection) error {
	var (
		pubKey    PublicKey
		sessionID SessionID
	)

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered")
	}

	if _, err := io.ReadFull(rand.Reader, state.privKey[:]); err != nil {
		return err
	}

	curve25519.ScalarBaseMult((*[32]byte)(&pubKey), (*[32]byte)(&state.privKey))

	if _, err := io.ReadFull(rand.Reader, state.salt[:]); err != nil {
		return err
	}

	serverPrivKey := []byte(os.Getenv("SERVER_ENCRYPT_PRIV_KEY"))
	sign := ed25519.Sign(serverPrivKey, append(pubKey[:], state.salt[:]...))

	if _, err := io.ReadFull(rand.Reader, sessionID[:]); err != nil {
		return err
	}
	state.encryptor.SetSessionID(sessionID)

	return state.writer.Write(connection, websocket.MessageText, NewInitMessage(i.ID, state.peerID, pubKey, sign, state.salt, sessionID))
}

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
