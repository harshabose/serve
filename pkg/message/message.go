package message

type Message interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

type Header struct {
	SenderID   string `json:"source_id"`
	ReceiverID string `json:"destination_id"`
}
