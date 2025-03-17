package log

import (
	"errors"
	"sync"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type manager struct {
	loggers map[interceptor.Connection]*logger
	mux     sync.RWMutex
}

func createManager() *manager {
	return &manager{
		loggers: make(map[interceptor.Connection]*logger),
	}
}

func (manager *manager) manage(connection interceptor.Connection) error {
	_, exists := manager.loggers[connection]
	if exists {
		return errors.New("logger already exists")
	}

	manager.loggers[connection] = &logger{}

	return nil
}

func (manager *manager) unmanage(connection interceptor.Connection) error {
	if logger, exists := manager.loggers[connection]; exists {
		logger.cleanup()
		delete(manager.loggers, connection)
		return nil
	}
	return errors.New("connection does not exists")
}

func (manager *manager) Process(msg message.Message, connection interceptor.Connection) error {
	logger, exists := manager.loggers[connection]
	if exists {
		return errors.New("logger already exists")
	}

	return logger.log(msg)
}

func (manager *manager) cleanup() {
	manager.mux.Lock()
	defer manager.mux.Unlock()

	for connection, logger := range manager.loggers {
		logger.cleanup()
		delete(manager.loggers, connection)
	}
}
