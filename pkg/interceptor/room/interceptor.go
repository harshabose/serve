package room

import (
	"context"
	"errors"
	"fmt"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type Interceptor struct {
	interceptor.NoOpInterceptor
	rooms  map[string]*room // map[roomID]room
	states map[interceptor.Connection]*state
}

func (i *Interceptor) BindSocketConnection(connection interceptor.Connection, writer interceptor.Writer, reader interceptor.Reader) error {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	if _, exists := i.states[connection]; exists {
		return errors.New("connection already exists")
	}

	i.states[connection] = &state{id: "unknown", writer: writer, reader: reader}

	return nil
}

func (i *Interceptor) InterceptSocketReader(reader interceptor.Reader) interceptor.Reader {
	return interceptor.ReaderFunc(func(connection interceptor.Connection) (websocket.MessageType, interceptor.Message, error) {
		messageType, data, err := reader.Read(connection)
		if err != nil {
			return messageType, data, err
		}

		msg, ok := data.(*interceptor.BaseMessage)
		if !ok || (msg.Protocol != interceptor.IProtocol && msg.MainType != MainType) {
			return messageType, data, nil
		}

		i.Mutex.Lock()
		defer i.Mutex.Unlock()

		if _, exists := i.states[connection]; exists {
			payload, err := PayloadUnmarshal(msg.SubType, msg.Payload)
			if err != nil {
				fmt.Println("error while processing room message: ", err.Error())
				return messageType, data, nil
			}

			if err := payload.Process(msg.Header, i, connection); err != nil {
				fmt.Println("error while processing room message: ", err.Error())
				return messageType, data, nil
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

	// for _, room := range i.rooms {
	// 	if room.owner == connection {
	// 		room.cancel()
	// 	}
	// }
}

func (i *Interceptor) Close() error {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	for _, room := range i.rooms {
		room.close()
	}
	i.rooms = make(map[string]*room)
	i.states = make(map[interceptor.Connection]*state)

	return nil
}

// ================================================================================================================== //
// ================================================================================================================== //

func (payload *CreateRoom) Process(header interceptor.Header, _interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	if err := payload.Validate(); err != nil {
		return err
	}

	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return errors.New("not appropriate interceptor to process this message")
	}

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	connState, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered yet")
	}

	connState.id = header.SenderID

	r, exists := i.rooms[payload.RoomID]
	if exists {
		fmt.Printf("room with ID '%s' already exists; trying to add client to the room instead\n", payload.RoomID)
		return r.add(connection, connState)
	}

	ctx, cancel := context.WithCancel(i.Ctx)
	r, err := newRoom(ctx, cancel, connection, connState, payload)
	if err != nil {
		return err
	}

	i.rooms[payload.RoomID] = r
	return nil
}

func (payload *JoinRoom) Process(header interceptor.Header, _interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	if err := payload.Validate(); err != nil {
		return err
	}

	i, ok := _interceptor.(*Interceptor)
	if !ok {
		return errors.New("not appropriate _interceptor to process this message")
	}

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered yet")
	}

	state.id = header.SenderID

	r, exists := i.rooms[payload.RoomID]
	if !exists {
		fmt.Printf("room with ID '%s' does not exists\n", payload.RoomID)
		return errors.New("room does not exists")
	}

	return r.add(connection, state)
}

func (payload *LeaveRoom) Process(header interceptor.Header, _interceptor interceptor.Interceptor, connection interceptor.Connection) error {
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

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered yet")
	}

	state.id = header.SenderID

	return r.remove(connection)
}

func (payload *ChatSource) Process(header interceptor.Header, _interceptor interceptor.Interceptor, connection interceptor.Connection) error {
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

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection not registered yet")
	}

	state.id = header.SenderID

	p := &ChatDest{RoomID: payload.RoomID, MessageID: payload.MessageID, Content: payload.Content, Timestamp: payload.Timestamp}
	return r.send(header.SenderID, p, payload.RecipientID...)
}
