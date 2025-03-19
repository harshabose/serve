package room

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type client struct {
	connection interceptor.Connection
	interceptor.WriterReader
}

type room struct {
	id           string
	owner        interceptor.Connection
	allowed      []string
	participants map[string]*client
	created      time.Time
	lastActivity time.Time
	ttl          time.Duration
	mux          sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
}

func (room *room) isAllowed(id string) bool {
	for _, allowed := range room.allowed {
		if id == allowed {
			return true
		}
	}

	return false
}

func (room *room) add(id string, connection interceptor.Connection, wr interceptor.WriterReader) error {
	room.mux.Lock()
	defer room.mux.Unlock()

	if !room.isAllowed(id) {
		return errors.New("participant not allowed")
	}

	if _, exists := room.participants[id]; exists {
		return errors.New("participant already exists")
	}

	room.participants[id] = &client{connection: connection, WriterReader: wr}
	room.lastActivity = time.Now()

	return nil
}

func (room *room) remove(id string, connection interceptor.Connection) error {
	room.mux.Lock()
	defer room.mux.Unlock()

	if room.owner == connection && connection != nil {
		fmt.Println("warn: room owner is being removed. this should not effect other functionalities until TTL")
		room.owner = nil
	}

	if id == "unknown" {
		if connection == nil {
			return errors.New("neither id nor connection are trackable to be used")
		}
		for testID, testConn := range room.participants {
			if testConn.connection == connection {
				return room.remove(testID, testConn.connection)
			}
		}
		return nil
	}

	if _, exists := room.participants[id]; !exists {
		return errors.New("participant does not exists")
	}
	delete(room.participants, id)

	for id, client := range room.participants {
		payload := &ClientLeft{RoomID: room.id, LeftAt: time.Now()}
		msg, err := CreateMessage("server", id, PayloadChatDestType, payload)
		if err != nil {
			fmt.Println("error while sending chat message to one of the recipient:", err.Error())
			continue
		}

		if err := client.Write(client.connection, websocket.MessageText, msg); err != nil {
			fmt.Println("error while sending chat message to one of the recipient:", err.Error())
			continue
		}
	}
	room.lastActivity = time.Now()

	return nil
}

func (room *room) send(senderID string, payload *ChatSource) error {
	room.mux.Lock()
	defer room.mux.Unlock()

	if len(payload.RecipientID) == 0 || payload.RecipientID == nil {
		payload.RecipientID = room.allowed
	}

	chat := &ChatDest{RoomID: payload.RoomID, MessageID: payload.MessageID, Content: payload.Content, Timestamp: payload.Timestamp}

	for _, receiverID := range payload.RecipientID {
		msg, err := CreateMessage(senderID, receiverID, PayloadChatDestType, chat)
		if err != nil {
			fmt.Println("error while sending chat message to one of the recipient:", err.Error())
			continue
		}

		client, exists := room.participants[receiverID]
		if !exists {
			fmt.Println("error while sending chat message to one of the recipient:", errors.New("participant does not exists").Error())
			continue
		}

		if err := client.Write(client.connection, websocket.MessageText, msg); err != nil {
			fmt.Println("error while sending chat message to one of the recipient:", errors.New("participant does not exists").Error())
			continue
		}
	}

	for id, client := range room.participants {
		payload := &ClientJoined{RoomID: room.id, JoinedAt: time.Now()}
		msg, err := CreateMessage("server", id, PayloadChatDestType, payload)
		if err != nil {
			fmt.Println("error while sending chat message to one of the recipient:", err.Error())
			continue
		}

		if err := client.Write(client.connection, websocket.MessageText, msg); err != nil {
			fmt.Println("error while sending chat message to one of the recipient:", err.Error())
			continue
		}
	}
	room.lastActivity = time.Now()

	return nil
}

func (room *room) close() {
	room.mux.Lock()
	defer room.mux.Unlock()

	room.cancel()
	room.owner = nil
	room.allowed = nil
	room.participants = make(map[string]*client)
}
