package log

import (
	"context"
	"io"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type Option = func(*Interceptor) error

type LoggerFactory struct {
	createFunc []func() (io.WriteCloser, error)
}

func CreateLoggerFactory() *LoggerFactory {
	return &LoggerFactory{
		createFunc: make([]func() (io.WriteCloser, error), 0),
	}
}

func (factory *LoggerFactory) Add(f func() (io.WriteCloser, error)) {
	factory.createFunc = append(factory.createFunc, f)
}

func (factory *LoggerFactory) Create() ([]io.WriteCloser, error) {
	loggers := make([]io.WriteCloser, 0)
	for _, createFunc := range factory.createFunc {
		logger, err := createFunc()
		if err != nil {
			return nil, err
		}
		loggers = append(loggers, logger)
	}

	return loggers, nil
}

type InterceptorFactory struct {
	opts []Option
}

func WithLoggerFactory(factory *LoggerFactory) Option {
	return func(i *Interceptor) error {
		i.loggerFactory = factory
		return nil
	}
}

func CreateInterceptorFactory(options ...Option) *InterceptorFactory {
	return &InterceptorFactory{
		opts: options,
	}
}

func (factory *InterceptorFactory) NewInterceptor(ctx context.Context, id string) (interceptor.Interceptor, error) {
	logInterceptor := &Interceptor{
		NoOpInterceptor: interceptor.NoOpInterceptor{
			ID:  id,
			Ctx: ctx,
		},
	}

	for _, option := range factory.opts {
		if err := option(logInterceptor); err != nil {
			return nil, err
		}
	}

	return logInterceptor, nil
}
