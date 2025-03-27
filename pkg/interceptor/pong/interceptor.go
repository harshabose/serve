package pong

import (
	"context"
	"errors"
	"fmt"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type Interceptor struct {
	interceptor.NoOpInterceptor
	states     map[interceptor.Connection]*state
	maxHistory uint16
}

func (i *Interceptor) BindSocketConnection(connection interceptor.Connection, writer interceptor.Writer, reader interceptor.Reader) error {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	_, exists := i.states[connection]
	if exists {
		return errors.New("owner already exists")
	}

	ctx, cancel := context.WithCancel(i.Ctx)

	i.states[connection] = &state{
		peerid: "unknown", // unknown until first ping
		writer: writer,    // full-stack writer (this is different from the writer in InterceptSocketWriter)
		reader: reader,
		pings:  make([]*ping, 0),
		pongs:  make([]*pong, 0),
		max:    i.maxHistory,
		ctx:    ctx,
		cancel: cancel,
	}

	return nil
}

func (i *Interceptor) InterceptSocketWriter(writer interceptor.Writer) interceptor.Writer {
	return interceptor.WriterFunc(func(conn interceptor.Connection, messageType websocket.MessageType, message interceptor.Message) error {
		i.Mutex.Lock()
		defer i.Mutex.Unlock()

		msg, ok := message.(*Message)
		if !ok {
			return writer.Write(conn, messageType, message)
		}

		payload := &Pong{}
		if err := payload.Unmarshal(msg.Payload); err != nil {
			return writer.Write(conn, messageType, message)
		}

		if _, exists := i.states[conn]; exists {
			if err := payload.Process(msg.Header, i, conn); err != nil {
				fmt.Println("error while processing ping pong message: ", err.Error())
			}
		}

		return writer.Write(conn, messageType, message)
	})
}

func (i *Interceptor) InterceptSocketReader(reader interceptor.Reader) interceptor.Reader {
	return interceptor.ReaderFunc(func(conn interceptor.Connection) (messageType websocket.MessageType, message interceptor.Message, err error) {
		messageType, message, err = reader.Read(conn)
		if err != nil {
			return messageType, message, err
		}

		i.Mutex.Lock()
		defer i.Mutex.Unlock()

		msg, ok := message.(*Message)
		if !ok {
			return messageType, message, nil
		}

		payload := &Ping{}
		if err := payload.Unmarshal(msg.Payload); err != nil {
			return messageType, message, nil
		}

		if _, exists := i.states[conn]; exists {
			if err := payload.Process(msg.Header, i, conn); err != nil {
				fmt.Println("error while processing ping pong message: ", err.Error())
			}
		}

		return messageType, message, nil
	})
}

func (i *Interceptor) UnBindSocketConnection(connection interceptor.Connection) {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	i.states[connection].cancel()
	delete(i.states, connection)
}

func (i *Interceptor) UnInterceptSocketWriter(_ interceptor.Writer) {
	// If left unimplemented, NoOpInterceptor's default implementation will be used
	// But, for reference, this method is implemented
}

func (i *Interceptor) UnInterceptSocketReader(_ interceptor.Reader) {
	// If left unimplemented, NoOpInterceptor's default implementation will be used
	// But, for reference, this method is implemented
}

func (i *Interceptor) Close() error {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	for _, state := range i.states {
		state.cancel()
		state.reader = nil
		state.writer = nil
	}
	i.states = make(map[interceptor.Connection]*state)

	return nil
}

func (payload *Ping) Process(header interceptor.Header, interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	if err := payload.Validate(); err != nil {
		return err
	}

	i, ok := interceptor.(*Interceptor)
	if !ok {
		return errors.New("not appropriate interceptor to process this message")
	}

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection does not exists")
	}

	state.peerid = header.SenderID
	state.recordPing(payload)

	return nil
}

func (payload *Pong) Process(_ interceptor.Header, interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	if err := payload.Validate(); err != nil {
		return err
	}

	i, ok := interceptor.(*Interceptor)
	if !ok {
		return errors.New("not appropriate interceptor to process this message")
	}

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection does not exists")
	}

	state.recordPong(payload)

	return nil
}
