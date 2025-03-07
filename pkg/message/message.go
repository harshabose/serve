package message

type BaseMessage interface {
	Marshal() ([]byte, error)
}

type Header struct {
	SourceID      string `json:"source_id"`
	DestinationID string `json:"destination_id"`
}

type Message struct {
	Header
}
