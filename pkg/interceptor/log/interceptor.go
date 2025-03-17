package log

import (
	"context"
	"errors"
	"fmt"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type Interceptor struct {
	interceptor.NoOpInterceptor
	manager *manager
}

func (log *Interceptor) BindSocketConnection(connection interceptor.Connection, writer interceptor.Writer, reader interceptor.Reader) error {
	log.Mutex.Lock()
	defer log.Mutex.Unlock()

	_, exists := log.State[connection]
	if exists {
		return errors.New("owner already exists")
	}

	ctx, cancel := context.WithCancel(log.Ctx)

	log.State[connection] = interceptor.State{
		ClientID: "unknown",
		Ctx:      ctx,
		Cancel:   cancel,
		Writer:   writer,
		Reader:   reader,
	}

	if err := log.manager.manage(connection); err != nil {
		return err
	}

	return nil
}

func (log *Interceptor) InterceptSocketWriter(writer interceptor.Writer) interceptor.Writer {
	return interceptor.WriterFunc(func(connection interceptor.Connection, messageType websocket.MessageType, message message.Message) error {
		log.Mutex.Lock() // TODO: CHECK IF MANUAL UNLOCK IS NEEDED RATHER THAN DEFER

		if err := log.manager.Process(message, connection); err != nil {
			fmt.Println("error while processing message in log interceptor:", err.Error())
		}

		log.Mutex.Unlock()
		return writer.Write(connection, messageType, message)
	})
}

func (log *Interceptor) InterceptSocketReader(reader interceptor.Reader) interceptor.Reader {
	return interceptor.ReaderFunc(func(connection interceptor.Connection) (websocket.MessageType, message.Message, error) {
		messageType, msg, err := reader.Read(connection)
		if err != nil {
			return messageType, msg, err
		}
		log.Mutex.Lock() // TODO: CHECK IF MANUAL UNLOCK IS NEEDED RATHER THAN DEFER

		if err := log.manager.Process(msg, connection); err != nil {
			fmt.Println("error while processing message in log interceptor:", err.Error())
		}

		log.Mutex.Unlock()
		return messageType, msg, err
	})
}

func (log *Interceptor) UnBindSocketConnection(_ interceptor.Connection) {

}

func (log *Interceptor) UnInterceptSocketWriter(_ interceptor.Writer) {

}

func (log *Interceptor) UnInterceptSocketReader(_ interceptor.Reader) {

}

func (log *Interceptor) Close() error {
	log.Mutex.Lock()
	defer log.Mutex.Unlock()

	for _, state := range log.State {
		state.Cancel()
		state.Reader = nil
		state.Writer = nil
	}
	log.State = make(map[interceptor.Connection]interceptor.State)
	log.manager.cleanup()

	return nil
}
