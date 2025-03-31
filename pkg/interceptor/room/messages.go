package room

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

var (
	MainType interceptor.MainType = "room"

	CreateRoomSubType       interceptor.SubType = "create_room"
	JoinRoomSubType         interceptor.SubType = "join_room"
	LeaveRoomSubType        interceptor.SubType = "leave_room"
	ChatSourceRoomSubType   interceptor.SubType = "chat_source"
	ChatDestRoomSubType     interceptor.SubType = "chat_destination"
	ClientJoinedRoomSubType interceptor.SubType = "client_joined"
	ClientLeftRoomSubType   interceptor.SubType = "client_left"
	SuccessRoomSubType      interceptor.SubType = "success"
	ErrorRoomSubType        interceptor.SubType = "error"

	subTypeMap = map[interceptor.SubType]interceptor.Payload{
		CreateRoomSubType:       &CreateRoom{},
		JoinRoomSubType:         &JoinRoom{},
		LeaveRoomSubType:        &LeaveRoom{},
		ChatSourceRoomSubType:   &ChatSource{},
		ChatDestRoomSubType:     &ChatDest{},
		ClientJoinedRoomSubType: &ClientJoined{},
		ClientLeftRoomSubType:   &ClientLeft{},
		SuccessRoomSubType:      &Success{},
		ErrorRoomSubType:        &Error{},
	}
)

func PayloadUnmarshal(sub interceptor.SubType, p json.RawMessage) error {
	if payload, exists := subTypeMap[sub]; exists {
		return payload.Unmarshal(p)
	}

	return errors.New("processor does not exist for given type")
}

func CreateMessage(senderID string, receiverID string, payload interceptor.Payload) (*interceptor.BaseMessage, error) {
	data, err := payload.Marshal()
	if err != nil {
		return nil, err
	}

	return &interceptor.BaseMessage{
		Header: interceptor.Header{
			SenderID:   senderID,
			ReceiverID: receiverID,
			Protocol:   interceptor.IProtocol,
			MainType:   MainType,
			SubType:    payload.Type(),
		},
		Payload: data,
	}, nil
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

func (payload *CreateRoom) Type() interceptor.SubType {
	return CreateRoomSubType
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

func (payload *JoinRoom) Type() interceptor.SubType {
	return JoinRoomSubType
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

func (payload *LeaveRoom) Type() interceptor.SubType {
	return LeaveRoomSubType
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

func (payload *ChatSource) Type() interceptor.SubType {
	return ChatSourceRoomSubType
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

func (payload *ChatDest) Process(_ interceptor.Header, _ interceptor.Interceptor, _ interceptor.Connection) error {
	return nil
}

func (payload *ChatDest) Type() interceptor.SubType {
	return ChatDestRoomSubType
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

func (payload *ClientJoined) Process(_ interceptor.Header, _ interceptor.Interceptor, _ interceptor.Connection) error {
	return nil
}

func (payload *ClientJoined) Type() interceptor.SubType {
	return ClientJoinedRoomSubType
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

func (payload *ClientLeft) Process(_ interceptor.Header, _ interceptor.Interceptor, _ interceptor.Connection) error {
	return nil
}

func (payload *ClientLeft) Type() interceptor.SubType {
	return ClientLeftRoomSubType
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

func (payload *Success) Process(_ interceptor.Header, _ interceptor.Interceptor, _ interceptor.Connection) error {
	return nil
}

func (payload *Success) Type() interceptor.SubType {
	return SuccessRoomSubType
}

// Specific success message creators

func JoinRoomSuccessMessage(clientID, roomID string) (*interceptor.BaseMessage, error) {
	payload := &Success{SuccessMessage: "Joined room " + roomID + " successfully"}
	return CreateMessage("server", clientID, payload)
}
func LeaveRoomSuccessMessage(clientID, roomID string) (*interceptor.BaseMessage, error) {
	payload := &Success{SuccessMessage: "Left room " + roomID + " successfully"}
	return CreateMessage("server", clientID, payload)
}

func ChatRoomSuccessMessage(clientID, messageID, roomID string) (*interceptor.BaseMessage, error) {
	payload := &Success{SuccessMessage: "message " + messageID + " " + roomID + " successfully"}
	return CreateMessage("server", clientID, payload)
}

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

func (payload *Error) Process(_ interceptor.Header, _ interceptor.Interceptor, _ interceptor.Connection) error {
	return nil
}

func (payload *Error) Type() interceptor.SubType {
	return ErrorRoomSubType
}

func CreateRoomErrorMessage(clientID, roomID string) (*interceptor.BaseMessage, error) {
	payload := &Error{ErrorMessage: "RoomMessage " + roomID + " created successfully"}
	return CreateMessage("server", clientID, payload)
}

func JoinRoomErrorMessage(clientID, roomID string) (*interceptor.BaseMessage, error) {
	payload := &Error{ErrorMessage: "Joined room " + roomID + " successfully"}
	return CreateMessage("server", clientID, payload)
}

func LeaveRoomErrorMessage(clientID, roomID string) (*interceptor.BaseMessage, error) {
	payload := &Error{ErrorMessage: "Left room " + roomID + " successfully"}
	return CreateMessage("server", clientID, payload)
}

func ChatRoomErrorMessage(clientID, messageID, roomID string) (*interceptor.BaseMessage, error) {
	payload := &Error{ErrorMessage: "message " + messageID + " " + roomID + " successfully"}
	return CreateMessage("server", clientID, payload)
}
