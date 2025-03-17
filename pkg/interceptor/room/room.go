package room

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type connection struct {
	id   string
	conn *websocket.Conn
	interceptor.Writer
	interceptor.Reader
}

type room struct {
	owner        *connection
	allowed      []string
	participants map[string]*connection
	created      time.Time
	lastActivity time.Time
	mux          sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
}

func createRoom(ctx context.Context, owner *connection, allowed []string, ttl time.Duration) *room {
	ctx2, cancel := context.WithTimeout(ctx, ttl)
	room := &room{
		owner:        owner,
		allowed:      allowed,
		participants: make(map[string]*connection),
		ctx:          ctx2,
		cancel:       cancel,
	}
	go room.loop()

	return room
}

func (room *room) isAllowed(clientID string) bool {
	if len(room.allowed) == 0 {
		return true
	}

	for _, id := range room.allowed {
		if id == clientID {
			return true
		}
	}

	if room.owner != nil && room.owner.id == clientID {
		return true
	}

	return false
}

func (room *room) add(conn *connection) error {
	room.mux.Lock()
	defer room.mux.Unlock()

	if !room.isAllowed(conn.id) {
		return errors.New("participant is not allowed")
	}

	if _, exists := room.participants[conn.id]; exists {
		return errors.New("participant already exists")
	}

	room.participants[conn.id] = conn
	room.lastActivity = time.Now()

	// TODO: Send ClientJoined to all participants
	return nil
}

func (room *room) remove(conn *connection) error {
	room.mux.Lock()
	defer room.mux.Unlock()

	if _, exists := room.participants[conn.id]; !exists {
		return errors.New("participant does not exists")
	}

	delete(room.participants, conn.id)
	room.lastActivity = time.Now()

	// TODO: Send ClientLeft to all participants
	return nil
}

func (room *room) send(sender *connection, participants []string, message []byte) error {
	if participants == nil || len(participants) == 0 {
		// broadcast
		for _, participant := range room.participants {
			participants = append(participants, participant.id)
		}
	}

	for _, participant := range participants {
		if connection, exists := room.participants[participant]; exists {
			data, err := CreateMessage(sender.id, participant, message).Marshal()
			if err != nil {
				return err
			}

			if err := connection.Write(connection.conn, websocket.MessageText, data); err != nil {
				return err
			}
		}
	}

	room.lastActivity = time.Now()
	return nil
}

func (room *room) loop() {
	defer room.close()

	select {
	case <-room.ctx.Done():
		// context cancelled
	}
	return
}

func (room *room) close() {
	room.mux.Lock()
	defer room.mux.Unlock()

	room.cancel()
}
