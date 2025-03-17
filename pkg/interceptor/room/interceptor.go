package room

import (
	"context"
	"fmt"
	"sync"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type Interceptor struct {
	interceptor.NoOpInterceptor
	roomsManager *manager
}

func (room *Interceptor) BindSocketConnection(connection *websocket.Conn, writer interceptor.Writer, reader interceptor.Reader) error {
	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	room.State[connection] = interceptor.State{Writer: writer, Reader: reader}

	return nil
}

func (room *Interceptor) BindSocketReader(reader interceptor.Reader) interceptor.Reader {
	return interceptor.ReaderFunc(func(conn *websocket.Conn) (websocket.MessageType, []byte, error) {
		messageType, data, err := reader.Read(conn)
		if err != nil {
			return messageType, data, err
		}

		msg := &Message{}
		if err := msg.Unmarshal(data); err != nil {
			return messageType, data, nil
		}
		room.Mutex.Lock()
		defer room.Mutex.Unlock()

		if connection, exists := room.State[conn]; exists {
			if err := room.roomsManager.Process(msg, connection.Writer, connection.Reader); err != nil {
				fmt.Println("error while processing message:", err.Error())
			}
		}

		return messageType, data, nil
	})
}

func (room *Interceptor) UnBindSocketConnection(connection *websocket.Conn, _ interceptor.Writer, _ interceptor.Reader) {

}

func (room *Interceptor) UnInterceptSocketWriter(writer interceptor.Writer) {

}

func (room *Interceptor) UnInterceptSocketReader(reader interceptor.Reader) {

}

func (room *Interceptor) Close() error {
	return nil
}
