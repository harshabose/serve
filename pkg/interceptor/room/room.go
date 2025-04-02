package room

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/utils"
)

type room struct {
	id           string
	owner        interceptor.Connection
	allowed      []string
	participants map[interceptor.Connection]*state
	created      time.Time
	lastActivity time.Time
	ttl          time.Duration
	mux          sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
}

func newRoom(ctx context.Context, cancel context.CancelFunc, connection interceptor.Connection, s *state, payload *CreateRoom) (*room, error) {
	r := &room{
		id:           payload.RoomID,
		owner:        connection,
		allowed:      payload.ClientsToAllow,
		participants: map[interceptor.Connection]*state{connection: s},
		created:      time.Now(),
		lastActivity: time.Now(),
		ttl:          payload.CloseTime,
		ctx:          ctx,
		cancel:       cancel,
	}

	if err := r.send("server", JoinRoomSuccessMessage(r.id), s.id); err != nil {
		return nil, err
	}

	return r, nil
}

func (room *room) isAllowed(id string) bool {
	for _, allowed := range room.allowed {
		if id == allowed {
			return true
		}
	}

	return false
}

func (room *room) add(connection interceptor.Connection, state *state) error {
	room.mux.Lock()
	defer room.mux.Unlock()

	merr := utils.NewMultiError()

	if !room.isAllowed(state.id) {
		return errors.New("participant not allowed")
	}

	if _, exists := room.participants[connection]; exists {
		return errors.New("participant already exists")
	}

	room.participants[connection] = state

	for _, client := range room.participants {
		if client.id != state.id {
			payload := &ClientJoined{ClientID: state.id, RoomID: room.id, JoinedAt: time.Now()}
			if err := room.send("server", payload, client.id); err != nil {
				merr.Add(err)
			}
		}
	}

	merr.Add(room.send("server", JoinRoomSuccessMessage(room.id), state.id))
	room.lastActivity = time.Now()

	return merr.ErrorOrNil()
}

func (room *room) send(from string, payload interceptor.Payload, to ...string) error {
	room.mux.Lock()
	defer room.mux.Unlock()

	merr := utils.NewMultiError()

	if len(to) == 0 || to == nil {
		to = room.allowed
	}

	for _, id := range to {
		msg, err := CreateMessage(from, id, payload)
		if err != nil {
			merr.Add(err)
		}

		if err := room.sendTo(id, msg); err != nil {
			merr.Add(err)
		}
	}

	return merr.ErrorOrNil()
}

func (room *room) sendTo(id string, msg *interceptor.BaseMessage) error {
	for conn, state := range room.participants {
		if state.id == id {
			return state.writer.Write(conn, websocket.MessageText, msg)
		}
	}

	return errors.New("connection does not exists")
}

func (room *room) remove(connection interceptor.Connection) error {
	room.mux.Lock()
	defer room.mux.Unlock()

	merr := utils.NewMultiError()

	if room.owner == connection && connection != nil {
		merr.Add(errors.New("warn: room owner is being removed. this should not effect other functionalities until TTL"))
		room.owner = nil
	}

	state, exists := room.participants[connection]
	if !exists {
		return errors.New("participant does not exists")
	}

	for _, client := range room.participants {
		payload := &ClientLeft{ClientID: state.id, RoomID: room.id, LeftAt: time.Now()}
		merr.Add(room.send("server", payload, client.id))
	}

	merr.Add(room.send("server", LeaveRoomSuccessMessage(room.id), state.id))

	delete(room.participants, connection)
	room.lastActivity = time.Now()

	return merr.ErrorOrNil()
}

func (room *room) close() {
	room.mux.Lock()
	defer room.mux.Unlock()

	room.cancel()
	room.owner = nil
	room.allowed = make([]string, 0)
	room.participants = make(map[interceptor.Connection]*state)
}

func (room *room) loop() {
	defer room.close()

	timer := time.NewTimer(room.ttl)
	defer timer.Stop()

	for {
		select {
		case <-room.ctx.Done():
			return
		case <-timer.C:
			return
		}
	}
}
