package encrypt

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type Interceptor struct {
	interceptor.NoOpInterceptor
	states map[interceptor.Connection]*state
	mux    sync.Mutex
	ctx    context.Context
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

		msg, err := state.encryptor.Encrypt(m.Message().SenderID, m.Message().ReceiverID, m)
		if err != nil {
			return writer.Write(connection, messageType, m)
		}

		return writer.Write(connection, messageType, msg)
	})
}

func (i *Interceptor) InterceptSocketReader(reader interceptor.Reader) interceptor.Reader {
	return interceptor.ReaderFunc(func(connection interceptor.Connection) (messageType websocket.MessageType, message message.Message, err error) {
		i.mux.Lock()
		defer i.mux.Unlock()

		messageType, message, err = reader.Read(connection)
		if err != nil {
			return messageType, message, err
		}

		msg, ok := message.(*interceptor.BaseMessage)
		if !ok || (msg.Protocol != interceptor.IProtocol && msg.MainType != MainType) {
			return messageType, message, nil
		}

		payload, err := PayloadUnmarshal(msg.SubType, msg.Payload)
		if err != nil {
			return messageType, message, err
		}

		state, exists := i.states[connection]
		if !exists {
			return messageType, message, nil
		}

		if err := payload.Process(msg.BaseMessage.Header, i, connection); err != nil {
			fmt.Println("error while processing encryptor message:", err.Error())
		}

		p, ok := payload.(*Encrypted)
		if !ok {
			return messageType, message, nil
		}

		message, err = state.encryptor.Decrypt(p)
		if err != nil {
			return messageType, message, nil
		}

		return messageType, message, nil
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
