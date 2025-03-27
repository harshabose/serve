package log

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type state struct {
	loggers []io.WriteCloser
	ctx     context.Context
	cancel  context.CancelFunc
	peerid  string
	mux     sync.RWMutex
}

func (state *state) cleanup() error {
	state.mux.Lock()
	defer state.mux.Unlock()

	for _, logger := range state.loggers {
		if err := logger.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (state *state) log(_ context.Context, msg interceptor.Message) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	state.mux.Lock()
	defer state.mux.Unlock()

	for _, logger := range state.loggers {
		if _, err := logger.Write(data); err != nil {
			fmt.Println("error while logging message:", err.Error())
			continue
		}
	}

	return nil
}
