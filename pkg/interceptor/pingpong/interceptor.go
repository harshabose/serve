package pingpong

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
	states     map[interceptor.Connection]*state
	maxHistory uint16
	interval   time.Duration // Time between ping messages
	ping       bool
}

func (i *Interceptor) BindSocketConnection(connection interceptor.Connection, writer interceptor.Writer, reader interceptor.Reader) error {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	_, exists := i.states[connection]
	if exists {
		return errors.New("connection already exists")
	}

	ctx, cancel := context.WithCancel(i.Ctx)

	i.states[connection] = &state{
		peerid: "unknown", // unknown until first pong
		writer: writer,    // full-stack writer (this is different from the writer in InterceptSocketWriter)
		reader: reader,
		pings:  make([]*ping, 0),
		pongs:  make([]*pong, 0),
		max:    i.maxHistory,
		ctx:    ctx,
		cancel: cancel,
	}

	if i.ping {
		go i.loop(ctx, i.interval, connection)
	}

	return nil
}

func (i *Interceptor) InterceptSocketWriter(writer interceptor.Writer) interceptor.Writer {
	return interceptor.WriterFunc(func(conn interceptor.Connection, messageType websocket.MessageType, m message.Message) error {
		i.Mutex.Lock()
		defer i.Mutex.Unlock()

		if _, exists := i.states[conn]; !exists {
			return writer.Write(conn, messageType, m)
		}

		payload, err := ProtocolUnmarshal(m.Message().Header.Protocol, m.Message().Payload)
		if err != nil {
			return writer.Write(conn, messageType, m)
		}

		if err := payload.Process(i, conn); err != nil {
			return writer.Write(conn, messageType, m)
		}

		return writer.Write(conn, messageType, m)
	})
}

func (i *Interceptor) InterceptSocketReader(reader interceptor.Reader) interceptor.Reader {
	return interceptor.ReaderFunc(func(conn interceptor.Connection) (messageType websocket.MessageType, m message.Message, err error) {
		messageType, m, err = reader.Read(conn)
		if err != nil {
			return messageType, m, err
		}

		i.Mutex.Lock()
		defer i.Mutex.Unlock()

		if _, exists := i.states[conn]; !exists {
			return messageType, m, nil
		}

		payload, err := ProtocolUnmarshal(m.Message().Header.Protocol, m.Message().Payload)
		if err != nil {
			return messageType, m, nil
		}

		if err := payload.Process(i, conn); err != nil {
			return messageType, m, nil
		}

		return messageType, m, nil
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

func (i *Interceptor) loop(ctx context.Context, interval time.Duration, connection interceptor.Connection) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state, exists := i.states[connection]
			if !exists {
				fmt.Println("error while trying to send ping:", errors.New("connection does not exists").Error())
				continue
			}

			msg, err := message.CreateMessage(i.ID, state.peerid, NewPing(i.ID, state.peerid))
			if err != nil {
				continue
			}

			if err := state.writer.Write(connection, websocket.MessageText, msg); err != nil {
				fmt.Println("error while trying to send ping:", err.Error())
				continue
			}
		}
	}
}

func (payload *Ping) Process(interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	if err := payload.Validate(); err != nil {
		return err
	}

	i := interceptor.(*Interceptor)

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection does not exists")
	}
	state.peerid = payload.SenderID
	state.recordPing(payload)

	if !i.ping {
		msg, err := message.CreateMessage(i.ID, state.peerid, NewPong(i.ID, payload))
		if err != nil {
			return err
		}
		return state.writer.Write(connection, websocket.MessageText, msg)
	}

	return nil

}

func (payload *Pong) Process(interceptor interceptor.Interceptor, connection interceptor.Connection) error {
	if err := payload.Validate(); err != nil {
		return err
	}

	i := interceptor.(*Interceptor)

	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	state, exists := i.states[connection]
	if !exists {
		return errors.New("connection does not exists")
	}

	state.peerid = payload.SenderID
	state.recordPong(payload)

	return nil
}
