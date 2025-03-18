package room

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type PayloadType string

const (
	PayloadCreateRoomType   PayloadType = "room:create_room"
	PayloadJoinRoomType     PayloadType = "room:join_room"
	PayloadLeaveRoomType    PayloadType = "room:leave_room"
	PayloadChatSourceType   PayloadType = "room:chat_sent"
	PayloadChatDestType     PayloadType = "room:chat_dest"
	PayloadClientJoinedType PayloadType = "room:client_joined"
	PayloadClientLeftType   PayloadType = "room:client_left"
	PayloadSuccessType      PayloadType = "room:success"
	PayloadErrorType        PayloadType = "room:error"
)

var payloadMap = map[PayloadType]interceptor.Payload{
	PayloadCreateRoomType:   &CreateRoom{},
	PayloadJoinRoomType:     &JoinRoom{},
	PayloadLeaveRoomType:    &LeaveRoom{},
	PayloadChatSourceType:   &ChatSource{},
	PayloadChatDestType:     &ChatDest{},
	PayloadClientJoinedType: &ClientJoined{},
	PayloadClientLeftType:   &ClientLeft{},
	PayloadSuccessType:      &Success{},
	PayloadErrorType:        &Error{},
}

func PayloadUnmarshal(_type PayloadType, p json.RawMessage) error {
	if payload, exists := payloadMap[_type]; exists {
		return payload.Unmarshal(p)
	}

	return errors.New("processor does not exist for given type")
}

type Message struct {
	message.Header
	Type    PayloadType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func CreateMessage(senderID string, receiverID string, payloadType PayloadType, payload interceptor.Payload) (*Message, error) {
	data, err := payload.Marshal()
	if err != nil {
		return nil, err
	}

	return &Message{
		Header: message.Header{
			SenderID:   senderID,
			ReceiverID: receiverID,
		},
		Type:    payloadType,
		Payload: data,
	}, nil
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

func (payload *CreateRoom) Marshal() ([]byte, error) {
	return json.Marshal(payload)
}

func (payload *CreateRoom) Unmarshal(data []byte) error {
	return json.Unmarshal(data, payload)
}

func (payload *CreateRoom) Validate() error {
	if payload.RoomID == "" || payload.CloseTime < 0 {
		return errors.New("not valid")
	}
	return nil
}

// JoinRoom is sent by clients to server to join an existing room
type JoinRoom struct {
	RoomID string `json:"room_id"`
}

func (payload *JoinRoom) Marshal() ([]byte, error) {
	return json.Marshal(payload)
}

func (payload *JoinRoom) Unmarshal(data []byte) error {
	return json.Unmarshal(data, payload)
}

func (payload *JoinRoom) Validate() error {
	if payload.RoomID == "" {
		return errors.New("not valid")
	}
	return nil
}

// LeaveRoom is sent by clients to server to leave a room
type LeaveRoom struct {
	RoomID string `json:"room_id"`
}

func (payload *LeaveRoom) Marshal() ([]byte, error) {
	return json.Marshal(payload)
}

func (payload *LeaveRoom) Unmarshal(data []byte) error {
	return json.Unmarshal(data, payload)
}

func (payload *LeaveRoom) Validate() error {
	if payload.RoomID == "" {
		return errors.New("not valid")
	}
	return nil
}

type ChatSource struct {
	RoomID      string          `json:"room_id"`
	MessageID   string          `json:"message_id"`
	RecipientID []string        `json:"recipient_id,omitempty"` // Empty for broadcast to room
	Content     json.RawMessage `json:"content"`
	Timestamp   time.Time       `json:"timestamp"`
}

func (payload *ChatSource) Marshal() ([]byte, error) {
	return json.Marshal(payload)
}

func (payload *ChatSource) Unmarshal(data []byte) error {
	return json.Unmarshal(data, payload)
}

func (payload *ChatSource) Validate() error {
	if payload.RoomID == "" || payload.MessageID == "" || payload.Content == nil {
		return errors.New("not valid")
	}
	return nil
}

type ChatDest struct {
	RoomID    string          `json:"room_id"`
	MessageID string          `json:"message_id"`
	Content   json.RawMessage `json:"content"`
	Timestamp time.Time       `json:"timestamp"`
}

func (payload *ChatDest) Marshal() ([]byte, error) {
	return json.Marshal(payload)
}

func (payload *ChatDest) Unmarshal(data []byte) error {
	return json.Unmarshal(data, payload)
}

func (payload *ChatDest) Validate() error {
	if payload.RoomID == "" || payload.MessageID == "" || payload.Content == nil {
		return errors.New("not valid")
	}
	return nil
}

func (payload *ChatDest) Process(_ message.Header, _ interceptor.Interceptor, _ interceptor.Connection) error {
	return nil
}

// ClientJoined is broadcast to room members when a new client joins
type ClientJoined struct {
	RoomID   string    `json:"room_id"`
	JoinedAt time.Time `json:"joined_at"`
}

func (payload *ClientJoined) Marshal() ([]byte, error) {
	return json.Marshal(payload)
}

func (payload *ClientJoined) Unmarshal(data []byte) error {
	return json.Unmarshal(data, payload)
}

func (payload *ClientJoined) Validate() error {
	if payload.RoomID == "" {
		return errors.New("not valid")
	}
	return nil
}

func (payload *ClientJoined) Process(_ message.Header, _ interceptor.Interceptor, _ interceptor.Connection) error {
	return nil
}

// ClientLeft is broadcast to room members when a client leaves
type ClientLeft struct {
	RoomID string    `json:"room_id"`
	LeftAt time.Time `json:"left_at"`
}

func (payload *ClientLeft) Marshal() ([]byte, error) {
	return json.Marshal(payload)
}

func (payload *ClientLeft) Unmarshal(data []byte) error {
	return json.Unmarshal(data, payload)
}

func (payload *ClientLeft) Validate() error {
	if payload.RoomID == "" {
		return errors.New("not valid")
	}
	return nil
}

func (payload *ClientLeft) Process(_ message.Header, _ interceptor.Interceptor, _ interceptor.Connection) error {
	return nil
}

// Success is sent to clients when a room operation succeeds
type Success struct {
	SuccessMessage string `json:"success_message"`
}

func (payload *Success) Marshal() ([]byte, error) {
	return json.Marshal(payload)
}

func (payload *Success) Unmarshal(data []byte) error {
	return json.Unmarshal(data, payload)
}

func (payload *Success) Validate() error {
	return nil
}

func (payload *Success) Process(_ message.Header, _ interceptor.Interceptor, _ interceptor.Connection) error {
	return nil
}

//
// // Specific success message creators
//
// func CreateRoomSuccessMessage(senderID, , roomID string) (*Message, error) {
// 	return CreateMessage()
// 	}
// }
// func JoinRoomSuccessMessage(clientID, roomID string) *Message {
// 	return &Message{
// 		Header: message.Header{
// 			SenderID:   "server",
// 			ReceiverID: clientID,
// 		},
// 		Payload: &Success{
// 			SuccessMessage: "Joined room " + roomID + " successfully",
// 		},
// 	}
// }
// func LeaveRoomSuccessMessage(clientID, roomID string) *Message {
// 	return &Message{
// 		Header: message.Header{
// 			SenderID:   "server",
// 			ReceiverID: clientID,
// 		},
// 		Payload: &Success{
// 			SuccessMessage: "Left room " + roomID + " successfully",
// 		},
// 	}
// }
// func ChatRoomSuccessMessage(clientID, messageID, roomID string) *Message {
// 	return &Message{
// 		Header: message.Header{
// 			SenderID:   "server",
// 			ReceiverID: clientID,
// 		},
// 		Payload: &Success{
// 			SuccessMessage: "message " + messageID + " " + roomID + " successfully",
// 		},
// 	}
// }
//

type Error struct {
	ErrorMessage string `json:"error_message"`
}

func (payload *Error) Marshal() ([]byte, error) {
	return json.Marshal(payload)
}

func (payload *Error) Unmarshal(data []byte) error {
	return json.Unmarshal(data, payload)
}

func (payload *Error) Validate() error {
	return nil
}

func (payload *Error) Process(_ message.Header, _ interceptor.Interceptor, _ interceptor.Connection) error {
	return nil
}

//
// func (msg *Error) Process(_ *manager, _ *websocket.Conn, _ interceptor.Writer) error {
// 	return nil
// }
//
// // Specific error message creators
//
// func CreateRoomErrorMessage(clientID, roomID string) *Message {
// 	return &Message{
// 		Header: message.Header{
// 			SenderID:   "server",
// 			ReceiverID: clientID,
// 		},
// 		Payload: &Error{
// 			ErrorMessage: "RoomMessage " + roomID + " created successfully",
// 		},
// 	}
// }
// func JoinRoomErrorMessage(clientID, roomID string) *Message {
// 	return &Message{
// 		Header: message.Header{
// 			SenderID:   "server",
// 			ReceiverID: clientID,
// 		},
// 		Payload: &Error{
// 			ErrorMessage: "Joined room " + roomID + " successfully",
// 		},
// 	}
// }
// func LeaveRoomErrorMessage(clientID, roomID string) *Message {
// 	return &Message{
// 		Header: message.Header{
// 			SenderID:   "server",
// 			ReceiverID: clientID,
// 		},
// 		Payload: &Error{
// 			ErrorMessage: "Left room " + roomID + " successfully",
// 		},
// 	}
// }
// func ChatRoomErrorMessage(clientID, messageID, roomID string) *Message {
// 	return &Message{
// 		Header: message.Header{
// 			SenderID:   "server",
// 			ReceiverID: clientID,
// 		},
// 		Payload: &Error{
// 			ErrorMessage: "message " + messageID + " " + roomID + " successfully",
// 		},
// 	}
// }
