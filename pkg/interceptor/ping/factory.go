package ping

import (
	"context"
	"time"
)

type Option = func(*Interceptor) error

type InterceptorFactory struct {
	opts []Option
}

func WithInterval(interval time.Duration) Option {
	return func(interceptor *Interceptor) error {
		interceptor.interval = interval
		return nil
	}
}

func WithStoreMax(max uint16) Option {
	return func(interceptor *Interceptor) error {
		interceptor.statsFactory.opts = append(interceptor.statsFactory.opts, withMax(max))
		return nil
	}
}

func CreateInterceptorFactory(options ...Option) *InterceptorFactory {
	return &InterceptorFactory{
		opts: options,
	}
}

func (factory *InterceptorFactory) NewInterceptor(ctx context.Context, id string) (*Interceptor, error) {
	pingInterceptor := &Interceptor{
		close:        make(chan struct{}),
		statsFactory: statsFactory{},
		ctx:          ctx,
	}

	for _, option := range factory.opts {
		if err := option(pingInterceptor); err != nil {
			return nil, err
		}
	}

	go pingInterceptor.loop()

	return pingInterceptor, nil
}
