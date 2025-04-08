package encrypt

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/coder/websocket"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/hkdf"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

var ServerPubKey = []byte(os.Getenv("SERVER_ENCRYPT_PUB_KEY"))

type Interceptor struct {
	interceptor.NoOpInterceptor
	states  map[interceptor.Connection]*state
	signKey []byte
	mux     sync.Mutex
	ctx     context.Context
}

func (i *Interceptor) BindSocketConnection(connection interceptor.Connection, writer interceptor.Writer, reader interceptor.Reader) error {
	i.mux.Lock()
	defer i.mux.Unlock()

	_, exists := i.states[connection]
	if exists {
		return errors.New("connection already exists")
	}

	ctx, cancel := context.WithCancel(i.Ctx)

	i.states[connection] = &state{
		id:        "unknown",
		encryptor: &aes256{},
		writer:    writer,
		reader:    reader,
		cancel:    cancel,
		ctx:       ctx,
	}

	// TODO: Exchange keys with the peer using a key exchange protocol like Diffie-Hellman
	// TODO: Store different keys for encryption and decryption

	return nil
}

func (i *Interceptor) InterceptSocketWriter(writer interceptor.Writer) interceptor.Writer {
	/*
		Takes in any type of message.Message and encrypts it. In general, all implementations of
		message.Message should use message.BaseMessage to implement message.Message.
	*/
	return interceptor.WriterFunc(func(connection interceptor.Connection, messageType websocket.MessageType, m message.Message) error {
		i.mux.Lock()
		defer i.mux.Unlock()

		state, exists := i.states[connection]
		if !exists {
			return writer.Write(connection, messageType, m)
		}

		encrypted, err := state.encryptor.Encrypt(m.Message().SenderID, m.Message().ReceiverID, m)
		if err != nil {
			return writer.Write(connection, messageType, m)
		}

		return writer.Write(connection, messageType, encrypted)
	})
}

func (i *Interceptor) InterceptSocketReader(reader interceptor.Reader) interceptor.Reader {
	return interceptor.ReaderFunc(func(connection interceptor.Connection) (websocket.MessageType, message.Message, error) {
		i.mux.Lock()
		defer i.mux.Unlock()

		messageType, m, err := reader.Read(connection)
		if err != nil {
			return messageType, m, err
		}

		state, exists := i.states[connection]
		if !exists {
			return messageType, m, nil
		}

		payload := &Encrypted{}
		if m.Message().Header.Protocol != payload.Protocol() {
			return messageType, m, nil
		}

		if err := payload.Unmarshal(m.Message().Payload); err != nil {
			return messageType, m, nil
		}

		if err := payload.Process(i, connection); err != nil {
			fmt.Println("error while processing encryptor m:", err.Error())
		}

		return messageType, payload.Message(), nil
	})
}

func (i *Interceptor) UnBindSocketConnection(connection interceptor.Connection) {
	i.mux.Lock()
	defer i.mux.Unlock()

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
	i.mux.Lock()
	defer i.mux.Unlock()

	i.states = make(map[interceptor.Connection]*state)

	return nil
}

func (i *Interceptor) exchangeKeys(connection interceptor.Connection) error {
	var privKey [32]byte
	var pubKey [32]byte

	if _, err := io.ReadFull(rand.Reader, privKey[:]); err != nil {
		return err
	}

	curve25519.ScalarBaseMult(&pubKey, &privKey)

	salt := make([]byte, 16)

	if _, err := io.ReadFull(rand.Reader, salt[:]); err != nil {
		return err
	}

	signature := append(pubKey[:], salt...)
	sign := ed25519.Sign(i.signKey, signature)

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered")
	}

	state.pubKey = pubKey[:]
	state.privKey = privKey[:]
	state.salt = salt

	return state.writer.Write(connection, websocket.MessageText, CreateEncryptionInit(i.ID, state.peerID, pubKey[:], sign, salt))
}

func derive(shared, salt []byte, info string) (encKey, decKey []byte, err error) {
	hkdfReader := hkdf.New(sha256.New, shared, salt, []byte(info))

	encKey = make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, encKey); err != nil {
		return nil, nil, err
	}

	decKey = make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, decKey); err != nil {
		return nil, nil, err
	}

	return encKey, decKey, nil
}
