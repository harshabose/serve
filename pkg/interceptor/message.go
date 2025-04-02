package interceptor

import (
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type (
	SubType  string
	MainType string
)

type Header struct {
	MainType MainType `json:"main_type"`
	SubType  SubType  `json:"sub_type"`
}

var IProtocol message.Protocol = "interceptor"

type BaseMessage struct {
	Header
	message.BaseMessage
}
