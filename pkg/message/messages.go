package message

import (
	"encoding/json"
	"time"
)

type Ping struct {
	Header
	Timestamp time.Time `json:"timestamp"`
}

func (ping *Ping) Marshal() ([]byte, error) {
	return json.Marshal(ping)
}

func (ping *Ping) Unmarshal(data []byte) error {
	return json.Unmarshal(data, ping)
}

func CreatePingMessage(timestamp time.Time) *Ping {
	return &Ping{
		Header: Header{
			SourceID:      "server",
			DestinationID: "unknown",
		},
		Timestamp: timestamp,
	}
}

type Pong struct {
	Header
	PingTimestamp time.Time `json:"ping_timestamp"`
	Timestamp     time.Time `json:"timestamp"`
}

func (pong *Pong) Marshal() ([]byte, error) {
	return json.Marshal(pong)
}

func (pong *Pong) Unmarshal(data []byte) error {
	return json.Unmarshal(data, pong)
}

func CreatePongMessage(pingTimestamp time.Time, timestamp time.Time) *Pong {
	return &Pong{
		Header: Header{
			SourceID:      "unknown",
			DestinationID: "server",
		},
		PingTimestamp: pingTimestamp,
		Timestamp:     timestamp,
	}
}
