package log

import (
	"context"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type Option = func(*Interceptor) error

type InterceptorFactory struct {
	opts []Option
}

// TODO: ADD OPTIONS
// TODO: ADD OPTIONS
// TODO: ADD OPTIONS
// TODO: ADD OPTIONS

// (Type of logging)
// DATABASE LOGGING
// IO (FILE) WRITER LOGGING
// STD OUT LOGGING

func CreateInterceptorFactory(options ...Option) *InterceptorFactory {
	return &InterceptorFactory{
		opts: options,
	}
}

func (factory *InterceptorFactory) NewInterceptor(ctx context.Context, id string) (interceptor.Interceptor, error) {
	logInterceptor := &Interceptor{
		NoOpInterceptor: interceptor.NoOpInterceptor{
			ID:    id,
			State: make(map[interceptor.Connection]interceptor.State),
			Ctx:   ctx,
		},
		manager: createManager(),
	}

	for _, option := range factory.opts {
		if err := option(logInterceptor); err != nil {
			return nil, err
		}
	}

	return logInterceptor, nil
}
