package encrypt

import (
	"context"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type Option = func(*Interceptor) error

func WithAES256(interceptor *Interceptor) error {

}

type InterceptorFactory struct {
	opts []Option
}

func (factory *InterceptorFactory) NewInterceptor(ctx context.Context, id string) (interceptor.Interceptor, error) {
	_interceptor := &Interceptor{
		NoOpInterceptor: interceptor.NoOpInterceptor{
			ID:  id,
			Ctx: ctx,
		},
	}

	for _, option := range factory.opts {
		if err := option(_interceptor); err != nil {
			return nil, err
		}
	}

	return _interceptor, nil
}
