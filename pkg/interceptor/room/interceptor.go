package room

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type Interceptor struct {
	interceptor.NoOpInterceptor
	rooms  map[string]*room // map[roomID]room
	states map[interceptor.Connection]interceptor.WriterReader
}

func (i *Interceptor) BindSocketConnection(connection interceptor.Connection, writer interceptor.Writer, reader interceptor.Reader) error {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	if _, exists := i.states[connection]; exists {
		return errors.New("connection already exists")
	}

	i.states[connection] = interceptor.WriterReader{Writer: writer, Reader: reader}

	return nil
}

func (i *Interceptor) InterceptSocketReader(reader interceptor.Reader) interceptor.Reader {
	return interceptor.ReaderFunc(func(connection interceptor.Connection) (websocket.MessageType, message.Message, error) {
		messageType, data, err := reader.Read(connection)
		if err != nil {
			return messageType, data, err
		}

		msg, ok := data.(*Message)
		if !ok {
			return messageType, data, nil
		}

		i.Mutex.Lock()
		defer i.Mutex.Unlock()

		if _, exists := i.states[connection]; exists {
			if err := PayloadUnmarshal(msg.Type, msg.Payload); err != nil {
				fmt.Println("error while processing room message: ", err.Error())
			}
		}

		return messageType, data, nil
	})
}

func (i *Interceptor) UnBindSocketConnection(connection interceptor.Connection) {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	if _, exists := i.states[connection]; exists {
		delete(i.states, connection)
	}

	for _, room := range i.rooms {
		if room.owner == connection {
			room.cancel()
		}
	}
}

func (i *Interceptor) Close() error {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	for _, room := range i.rooms {
		room.close()
	}
	i.rooms = make(map[string]*room)
	i.states = make(map[interceptor.Connection]interceptor.WriterReader)

	return nil
}

// ================================================================================================================== //
// ================================================================================================================== //

func (payload *CreateRoom) Process(header message.Header, _interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	if err := payload.Validate(); err != nil {
		return err
	}

	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return errors.New("not appropriate _interceptor to process this message")
	}

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	wr, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered yet")
	}

	r, exists := i.rooms[payload.RoomID]
	if exists {
		fmt.Printf("room with ID '%s' already exists; trying to add to the room instead\n", payload.RoomID)
		return r.add(header.SenderID, connection, wr)
	}

	ctx, cancel := context.WithCancel(i.Ctx)
	r = &room{
		id:           payload.RoomID,
		owner:        connection,
		allowed:      payload.ClientsToAllow,
		participants: map[string]*client{header.SenderID: {connection: connection, WriterReader: wr}},
		created:      time.Now(),
		lastActivity: time.Now(),
		ttl:          payload.CloseTime,
		ctx:          ctx,
		cancel:       cancel,
	}
	i.rooms[payload.RoomID] = r

	return nil
}

func (payload *JoinRoom) Process(header message.Header, _interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	if err := payload.Validate(); err != nil {
		return err
	}

	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return errors.New("not appropriate _interceptor to process this message")
	}

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	wr, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered yet")
	}

	r, exists := i.rooms[payload.RoomID]
	if !exists {
		fmt.Printf("room with ID '%s' does not exists\n", payload.RoomID)
		return errors.New("room does not exists")
	}

	return r.add(header.SenderID, connection, wr)
}

func (payload *LeaveRoom) Process(header message.Header, _interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	if err := payload.Validate(); err != nil {
		return err
	}

	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return errors.New("not appropriate _interceptor to process this message")
	}

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	r, exists := i.rooms[payload.RoomID]
	if !exists {
		fmt.Printf("room with ID '%s' does not exists\n", payload.RoomID)
		return errors.New("room does not exists")
	}

	return r.remove(header.SenderID, connection)
}

func (payload *ChatSource) Process(header message.Header, _interceptor interceptor.Interceptor, _ interceptor.Connection) error {
	if err := payload.Validate(); err != nil {
		return err
	}

	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return errors.New("not appropriate _interceptor to process this message")
	}

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	r, exists := i.rooms[payload.RoomID]
	if !exists {
		fmt.Printf("room with ID '%s' does not exists\n", payload.RoomID)
		return errors.New("room does not exists")
	}

	return r.send(header.SenderID, payload)
}
