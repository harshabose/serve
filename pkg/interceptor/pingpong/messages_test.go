package pingpong

import (
	_ "bytes"
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

func TestPing_MarshalUnmarshal(t *testing.T) {
	now := time.Now()
	ping := &Ping{
		MessageID: "test-iamserver-123",
		Timestamp: now,
	}

	data, err := ping.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaledPing Ping
	err = unmarshaledPing.Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaledPing.MessageID != ping.MessageID {
		t.Errorf("MessageID mismatch: got %v, want %v", unmarshaledPing.MessageID, ping.MessageID)
	}
	if !unmarshaledPing.Timestamp.Equal(ping.Timestamp) {
		t.Errorf("Timestamp mismatch: got %v, want %v", unmarshaledPing.Timestamp, ping.Timestamp)
	}
}

func TestPing_Validate(t *testing.T) {
	ping := &Ping{
		MessageID: "test-iamserver-123",
		Timestamp: time.Now(),
	}

	err := ping.Validate()
	if err != nil {
		t.Errorf("Validate failed unexpectedly: %v", err)
	}
}

func TestPong_MarshalUnmarshal(t *testing.T) {
	pingTime := time.Now().Add(-time.Second)
	pongTime := time.Now()
	pong := &Pong{
		MessageID:     "test-iamserver-123",
		PingTimestamp: pingTime,
		Timestamp:     pongTime,
	}

	data, err := pong.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaledPong Pong
	err = unmarshaledPong.Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaledPong.MessageID != pong.MessageID {
		t.Errorf("MessageID mismatch: got %v, want %v", unmarshaledPong.MessageID, pong.MessageID)
	}
	if !unmarshaledPong.PingTimestamp.Equal(pong.PingTimestamp) {
		t.Errorf("PingTimestamp mismatch: got %v, want %v", unmarshaledPong.PingTimestamp, pong.PingTimestamp)
	}
	if !unmarshaledPong.Timestamp.Equal(pong.Timestamp) {
		t.Errorf("Timestamp mismatch: got %v, want %v", unmarshaledPong.Timestamp, pong.Timestamp)
	}
}

func TestPong_Validate(t *testing.T) {
	pong := &Pong{
		MessageID:     "test-iamserver-123",
		PingTimestamp: time.Now().Add(-time.Second),
		Timestamp:     time.Now(),
	}

	err := pong.Validate()
	if err != nil {
		t.Errorf("Validate failed unexpectedly: %v", err)
	}
}

func TestMessage_MarshalUnmarshalWithPingPayload(t *testing.T) {
	now := time.Now()
	pingPayload := &Ping{
		MessageID: "test-iamserver-123",
		Timestamp: now,
	}
	senderID := "server"
	receiverID := "client"

	msg, err := CreateMessage(senderID, receiverID, pingPayload)
	if err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}

	data, err := msg.Marshal()
	if err != nil {
		t.Fatalf("Message Marshal failed: %v", err)
	}

	var unmarshaledMsg Message
	err = unmarshaledMsg.Unmarshal(data)
	if err != nil {
		t.Fatalf("Message Unmarshal failed: %v", err)
	}

	if unmarshaledMsg.Header.SenderID != senderID {
		t.Errorf("SenderID mismatch: got %v, want %v", unmarshaledMsg.Header.SenderID, senderID)
	}
	if unmarshaledMsg.Header.ReceiverID != receiverID {
		t.Errorf("ReceiverID mismatch: got %v, want %v", unmarshaledMsg.Header.ReceiverID, receiverID)
	}

	var unmarshaledPayload Ping
	err = json.Unmarshal(unmarshaledMsg.Payload, &unmarshaledPayload)
	if err != nil {
		t.Fatalf("Unmarshal payload failed: %v", err)
	}

	if unmarshaledPayload.MessageID != pingPayload.MessageID {
		t.Errorf("Payload MessageID mismatch: got %v, want %v", unmarshaledPayload.MessageID, pingPayload.MessageID)
	}
	if !unmarshaledPayload.Timestamp.Equal(pingPayload.Timestamp) {
		t.Errorf("Payload Timestamp mismatch: got %v, want %v", unmarshaledPayload.Timestamp, pingPayload.Timestamp)
	}
}

func TestMessage_MarshalUnmarshalWithPongPayload(t *testing.T) {
	pingTime := time.Now().Add(-time.Second)
	pongTime := time.Now()
	pongPayload := &Pong{
		MessageID:     "test-iamserver-123",
		PingTimestamp: pingTime,
		Timestamp:     pongTime,
	}
	senderID := "client"
	receiverID := "server"

	msg, err := CreateMessage(senderID, receiverID, pongPayload)
	if err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}

	data, err := msg.Marshal()
	if err != nil {
		t.Fatalf("Message Marshal failed: %v", err)
	}

	var unmarshaledMsg Message
	err = unmarshaledMsg.Unmarshal(data)
	if err != nil {
		t.Fatalf("Message Unmarshal failed: %v", err)
	}

	if unmarshaledMsg.Header.SenderID != senderID {
		t.Errorf("SenderID mismatch: got %v, want %v", unmarshaledMsg.Header.SenderID, senderID)
	}
	if unmarshaledMsg.Header.ReceiverID != receiverID {
		t.Errorf("ReceiverID mismatch: got %v, want %v", unmarshaledMsg.Header.ReceiverID, receiverID)
	}

	var unmarshaledPayload Pong
	err = json.Unmarshal(unmarshaledMsg.Payload, &unmarshaledPayload)
	if err != nil {
		t.Fatalf("Unmarshal payload failed: %v", err)
	}

	if unmarshaledPayload.MessageID != pongPayload.MessageID {
		t.Errorf("Payload MessageID mismatch: got %v, want %v", unmarshaledPayload.MessageID, pongPayload.MessageID)
	}
	if !unmarshaledPayload.PingTimestamp.Equal(pongPayload.PingTimestamp) {
		t.Errorf("Payload PingTimestamp mismatch: got %v, want %v", unmarshaledPayload.PingTimestamp, pongPayload.PingTimestamp)
	}
	if !unmarshaledPayload.Timestamp.Equal(pongPayload.Timestamp) {
		t.Errorf("Payload Timestamp mismatch: got %v, want %v", unmarshaledPayload.Timestamp, pongPayload.Timestamp)
	}
}

func TestCreateMessage(t *testing.T) {
	pingPayload := &Ping{
		MessageID: "test-iamserver-123",
		Timestamp: time.Now(),
	}
	senderID := "test-sender"
	receiverID := "test-receiver"

	msg, err := CreateMessage(senderID, receiverID, pingPayload)
	if err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}

	if msg.Header.SenderID != senderID {
		t.Errorf("CreateMessage SenderID mismatch: got %v, want %v", msg.Header.SenderID, senderID)
	}
	if msg.Header.ReceiverID != receiverID {
		t.Errorf("CreateMessage ReceiverID mismatch: got %v, want %v", msg.Header.ReceiverID, receiverID)
	}

	var unmarshaledPayload Ping
	err = json.Unmarshal(msg.Payload, &unmarshaledPayload)
	if err != nil {
		t.Fatalf("Unmarshal payload failed: %v", err)
	}

	if unmarshaledPayload.MessageID != pingPayload.MessageID {
		t.Errorf("CreateMessage Payload MessageID mismatch: got %v, want %v", unmarshaledPayload.MessageID, pingPayload.MessageID)
	}
	if !unmarshaledPayload.Timestamp.Equal(pingPayload.Timestamp) {
		t.Errorf("CreateMessage Payload Timestamp mismatch: got %v, want %v", unmarshaledPayload.Timestamp, pingPayload.Timestamp)
	}
}

// MockInterceptor for testing Process methods
type MockInterceptor struct {
	states map[interface{}]*MockState
	Mutex  sync.Mutex
}

func NewMockInterceptor() *MockInterceptor {
	return &MockInterceptor{
		states: make(map[interface{}]*MockState),
	}
}

func (m *MockInterceptor) BindSocketConnection(_ interceptor.Connection, _ interceptor.Writer, _ interceptor.Reader) error {
	return nil
}
func (m *MockInterceptor) InterceptSocketWriter(writer interceptor.Writer) interceptor.Writer {
	return writer
}
func (m *MockInterceptor) InterceptSocketReader(reader interceptor.Reader) interceptor.Reader {
	return reader
}
func (m *MockInterceptor) UnBindSocketConnection(_ interceptor.Connection) {}
func (m *MockInterceptor) UnInterceptSocketWriter(_ interceptor.Writer)    {}
func (m *MockInterceptor) UnInterceptSocketReader(_ interceptor.Reader)    {}
func (m *MockInterceptor) Close() error                                    { return nil }

// MockConnection for testing Process methods
type MockConnection struct {
	ID string
}

func (m *MockConnection) Write(_ context.Context, _ []byte) error {
	return nil
}

func (m *MockConnection) Read(_ context.Context) ([]byte, error) {
	return nil, nil
}

// MockState for tracking iamserver/pong records
type MockState struct {
	pings  []*Ping
	pongs  []*Pong
	peerid string
}

func (m *MockInterceptor) GetState(conn *MockConnection) *MockState {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()
	if _, exists := m.states[conn]; !exists {
		m.states[conn] = &MockState{}
	}
	return m.states[conn]
}

func (s *MockState) recordPing(ping *Ping) {
	s.pings = append(s.pings, ping)
}

func (s *MockState) recordPong(pong *Pong) {
	s.pongs = append(s.pongs, pong)
}
