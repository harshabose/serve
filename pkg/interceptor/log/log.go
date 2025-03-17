package log

import (
	"fmt"
	"io"

	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type logger struct {
	writers []io.Writer
}

func (logger *logger) cleanup() {

}

func (logger *logger) log(msg message.Message) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	for _, writer := range logger.writers {
		if _, err := writer.Write(data); err != nil {
			fmt.Println("error while logging message:", err.Error())
			continue
		}
	}
	return nil
}
