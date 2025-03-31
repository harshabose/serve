package message

type Message interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}
