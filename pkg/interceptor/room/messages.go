package room

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type Payload interface {
	message.Message
	Validate() error
	Process(message.Header, *manager, *websocket.Conn, interceptor.Writer, interceptor.Reader) error
}

type Message struct {
	message.Header
	Payload Payload `json:"payload"`
}

func CreateMessage(senderID string, receiverID string, msg Payload) *Message {
	return &Message{
		Header: message.Header{
			SenderID:   senderID,
			ReceiverID: receiverID,
		},
		Payload: msg,
	}
}

func (msg *Message) Marshal() ([]byte, error) {
	return json.Marshal(msg)
}

func (msg *Message) Unmarshal(data []byte) error {
	return json.Unmarshal(data, msg)
}

type CreateRoom struct {
	RoomID         string        `json:"room_id"`
	CloseTime      time.Duration `json:"close_time"`
	ClientsToAllow []string      `json:"clients_to_allow"`
}

func (msg *CreateRoom) Marshal() ([]byte, error) {
	return json.Marshal(msg)
}

func (msg *CreateRoom) Unmarshal(data []byte) error {
	return json.Unmarshal(data, msg)
}

func (msg *CreateRoom) Validate() error {
	if msg.RoomID == "" || msg.CloseTime < 0 {
		return errors.New("not valid")
	}
	return nil
}

// JoinRoom is sent by clients to server to join an existing room
type JoinRoom struct {
	ClientID string `json:"client_id"`
	RoomID   string `json:"room_id"`
}

func (msg *JoinRoom) Marshal() ([]byte, error) {
	return json.Marshal(msg)
}

func (msg *JoinRoom) Unmarshal(data []byte) error {
	return json.Unmarshal(data, msg)
}

func (msg *JoinRoom) Validate() error {
	if msg.RoomID == "" {
		return errors.New("not valid")
	}
	return nil
}

// LeaveRoom is sent by clients to server to leave a room
type LeaveRoom struct {
	ClientID string `json:"client_id"`
	RoomID   string `json:"room_id"`
}

func (msg *LeaveRoom) Marshal() ([]byte, error) {
	return json.Marshal(msg)
}

func (msg *LeaveRoom) Unmarshal(data []byte) error {
	return json.Unmarshal(data, msg)
}

func (msg *LeaveRoom) Validate() error {
	if msg.ClientID == "" || msg.RoomID == "" {
		return errors.New("not valid")
	}
	return nil
}

type Chat struct {
	ClientID    string    `json:"client_id"`
	RoomID      string    `json:"room_id"`
	MessageID   string    `json:"message_id"`
	RecipientID []string  `json:"recipient_id,omitempty"` // Empty for broadcast to room
	Content     []byte    `json:"content"`
	Timestamp   time.Time `json:"timestamp"`
}

func (msg *Chat) Marshal() ([]byte, error) {
	return json.Marshal(msg)
}

func (msg *Chat) Unmarshal(data []byte) error {
	return json.Unmarshal(data, msg)
}

func (msg *Chat) Validate() error {
	if msg.ClientID == "" || msg.RoomID == "" || msg.MessageID == "" || msg.Content == nil {
		return errors.New("not valid")
	}
	return nil
}

// ClientJoined is broadcast to room members when a new client joins
type ClientJoined struct {
	RoomID   string    `json:"room_id"`
	ClientID string    `json:"client_id"`
	JoinedAt time.Time `json:"joined_at"`
}

func (msg *ClientJoined) Marshal() ([]byte, error) {
	return json.Marshal(msg)
}

func (msg *ClientJoined) Unmarshal(data []byte) error {
	return json.Unmarshal(data, msg)
}

func (msg *ClientJoined) Validate() error {
	if msg.RoomID == "" || msg.ClientID == "" {
		return errors.New("not valid")
	}
	return nil
}

func (msg *ClientJoined) Process(_ *manager, _ *websocket.Conn, _ interceptor.Writer) error {
	return nil
}

// ClientLeft is broadcast to room members when a client leaves
type ClientLeft struct {
	RoomID   string    `json:"room_id"`
	ClientID string    `json:"client_id"`
	LeftAt   time.Time `json:"left_at"`
}

func (msg *ClientLeft) Marshal() ([]byte, error) {
	return json.Marshal(msg)
}

func (msg *ClientLeft) Unmarshal(data []byte) error {
	return json.Unmarshal(data, msg)
}

func (msg *ClientLeft) Validate() error {
	if msg.RoomID == "" || msg.ClientID == "" {
		return errors.New("not valid")
	}
	return nil
}

func (msg *ClientLeft) Process(_ *manager, _ *websocket.Conn, _ interceptor.Writer) error {
	return nil
}

// Success is sent to clients when a room operation succeeds
type Success struct {
	SuccessMessage string `json:"success_message"`
}

func (msg *Success) Marshal() ([]byte, error) {
	return json.Marshal(msg)
}

func (msg *Success) Unmarshal(data []byte) error {
	return json.Unmarshal(data, msg)
}

func (msg *Success) Validate() error {
	return nil
}

func (msg *Success) Process(_ *manager, _ *websocket.Conn, _ interceptor.Writer) error {
	return nil
}

// Specific success message creators

func CreateRoomSuccessMessage(clientID, roomID string) *Message {
	return &Message{
		Header: message.Header{
			SenderID:   "server",
			ReceiverID: clientID,
		},
		Payload: &Success{
			SuccessMessage: "RoomMessage " + roomID + " created successfully",
		},
	}
}
func JoinRoomSuccessMessage(clientID, roomID string) *Message {
	return &Message{
		Header: message.Header{
			SenderID:   "server",
			ReceiverID: clientID,
		},
		Payload: &Success{
			SuccessMessage: "Joined room " + roomID + " successfully",
		},
	}
}
func LeaveRoomSuccessMessage(clientID, roomID string) *Message {
	return &Message{
		Header: message.Header{
			SenderID:   "server",
			ReceiverID: clientID,
		},
		Payload: &Success{
			SuccessMessage: "Left room " + roomID + " successfully",
		},
	}
}
func ChatRoomSuccessMessage(clientID, messageID, roomID string) *Message {
	return &Message{
		Header: message.Header{
			SenderID:   "server",
			ReceiverID: clientID,
		},
		Payload: &Success{
			SuccessMessage: "message " + messageID + " " + roomID + " successfully",
		},
	}
}

type Error struct {
	ErrorMessage string `json:"error_message"`
}

func (msg *Error) Marshal() ([]byte, error) {
	return json.Marshal(msg)
}

func (msg *Error) Unmarshal(data []byte) error {
	return json.Unmarshal(data, msg)
}

func (msg *Error) Validate() error {
	return nil
}

func (msg *Error) Process(_ *manager, _ *websocket.Conn, _ interceptor.Writer) error {
	return nil
}

// Specific error message creators

func CreateRoomErrorMessage(clientID, roomID string) *Message {
	return &Message{
		Header: message.Header{
			SenderID:   "server",
			ReceiverID: clientID,
		},
		Payload: &Error{
			ErrorMessage: "RoomMessage " + roomID + " created successfully",
		},
	}
}
func JoinRoomErrorMessage(clientID, roomID string) *Message {
	return &Message{
		Header: message.Header{
			SenderID:   "server",
			ReceiverID: clientID,
		},
		Payload: &Error{
			ErrorMessage: "Joined room " + roomID + " successfully",
		},
	}
}
func LeaveRoomErrorMessage(clientID, roomID string) *Message {
	return &Message{
		Header: message.Header{
			SenderID:   "server",
			ReceiverID: clientID,
		},
		Payload: &Error{
			ErrorMessage: "Left room " + roomID + " successfully",
		},
	}
}
func ChatRoomErrorMessage(clientID, messageID, roomID string) *Message {
	return &Message{
		Header: message.Header{
			SenderID:   "server",
			ReceiverID: clientID,
		},
		Payload: &Error{
			ErrorMessage: "message " + messageID + " " + roomID + " successfully",
		},
	}
}
