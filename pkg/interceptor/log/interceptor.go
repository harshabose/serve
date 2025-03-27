package log

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type Interceptor struct {
	interceptor.NoOpInterceptor
	loggerFactory *LoggerFactory
	states        map[interceptor.Connection]*state
}

func (i *Interceptor) BindSocketConnection(connection interceptor.Connection, _ interceptor.Writer, _ interceptor.Reader) error {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	_, exists := i.states[connection]
	if exists {
		return errors.New("connection already exists")
	}

	loggers, err := i.loggerFactory.Create()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(i.Ctx)

	i.states[connection] = &state{
		loggers: loggers,
		peerid:  "unknown",
		ctx:     ctx,
		cancel:  cancel,
	}

	return nil
}

func (i *Interceptor) InterceptSocketWriter(writer interceptor.Writer) interceptor.Writer {
	return interceptor.WriterFunc(func(connection interceptor.Connection, messageType websocket.MessageType, message interceptor.Message) error {
		i.Mutex.Lock()

		state, exists := i.states[connection]
		if !exists {
			return errors.New("connection does not exists")
		}

		ctx, cancel := context.WithTimeout(state.ctx, time.Second)

		if err := state.log(ctx, message); err != nil {
			cancel()
			return err
		}

		cancel()
		i.Mutex.Unlock()
		return writer.Write(connection, messageType, message)
	})
}

func (i *Interceptor) InterceptSocketReader(reader interceptor.Reader) interceptor.Reader {
	return interceptor.ReaderFunc(func(connection interceptor.Connection) (messageType websocket.MessageType, message interceptor.Message, err error) {
		messageType, message, err = reader.Read(connection)
		if err != nil {
			return messageType, message, err
		}
		i.Mutex.Lock()

		state, exists := i.states[connection]
		if !exists {
			return messageType, message, err
		}

		ctx, cancel := context.WithTimeout(state.ctx, time.Second)

		if err := state.log(ctx, message); err != nil {
			cancel()
			return messageType, message, err
		}

		cancel()
		i.Mutex.Unlock()
		return messageType, message, err
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
	if err := state.cleanup(); err != nil {
		fmt.Println("error while unbinding connection:", err.Error())
		return
	}
	delete(i.states, connection)

	return
}

func (i *Interceptor) Close() error {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	for _, state := range i.states {
		state.cancel()
		if err := state.cleanup(); err != nil {
			return err
		}
	}
	i.states = make(map[interceptor.Connection]*state)

	return nil
}
