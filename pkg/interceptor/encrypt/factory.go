package encrypt

import (
	"context"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

// Option defines a function type that configures an Interceptor instance
type Option = func(*Interceptor) error

// WithAES256 configures the interceptor to use AES-256 encryption
func WithAES256(interceptor *Interceptor) error {
	// Initialize the encryptor with AES-256 implementation for new connections
	interceptor.encryptorFactor = NewAES256
	return nil
}

// WithServer marks this interceptor as a server-side interceptor
// Server-side interceptors have different behavior for session handling
func WithServer(interceptor *Interceptor) error {
	interceptor.isServer = true
	return nil
}

// InterceptorFactory creates encryption interceptors with configured options
type InterceptorFactory struct {
	opts []Option
}

// CreateInterceptorFactory constructs a new factory with the provided options
func CreateInterceptorFactory(options ...Option) *InterceptorFactory {
	return &InterceptorFactory{
		opts: options,
	}
}

// NewInterceptor creates and configures a new encryption interceptor
// Implements the interceptor.Factory interface
func (factory *InterceptorFactory) NewInterceptor(ctx context.Context, id string) (interceptor.Interceptor, error) {
	// initialiseKeyExchange() // TODO: For some reason, this function is hidden
	_interceptor := &Interceptor{
		NoOpInterceptor: interceptor.NoOpInterceptor{
			ID:  id,
			Ctx: ctx,
		},
		states:          make(map[interceptor.Connection]*state),
		isServer:        false,
		encryptorFactor: NewAES256,
	}

	// Apply all configured options
	for _, option := range factory.opts {
		if err := option(_interceptor); err != nil {
			return nil, err
		}
	}

	return _interceptor, nil
}
