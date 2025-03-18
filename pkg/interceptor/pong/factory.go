package pong

import (
	"context"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

// Option defines a function type that configures an Interceptor instance.
// Each option modifies a specific aspect of the interceptor's behavior
// and returns an error if the configuration cannot be applied.
type Option = func(*Interceptor) error

// InterceptorFactory creates ping interceptors with a predefined set of options.
// It implements the interceptor.Factory interface, allowing it to be registered
// with the interceptor registry for automatic interceptor creation.
type InterceptorFactory struct {
	opts []Option // Collection of configuration options to apply
}

// WithMaxHistory creates an option that sets the maximum number of ping/pong
// records to keep in history. This limits memory usage while still allowing
// for statistical analysis of connection performance.
//
// Parameters:
//   - max: Maximum number of historical ping/pong records to maintain
//
// Returns:
//   - An Option that configures history limit when applied to an interceptor
func WithMaxHistory(max uint16) Option {
	return func(interceptor *Interceptor) error {
		interceptor.maxHistory = max
		return nil
	}
}

// CreateInterceptorFactory constructs a new factory that will create ping interceptors
// with the provided options. The options are stored and applied to each new
// interceptor created by the factory.
//
// Parameters:
//   - options: Variable number of options to configure created interceptors
//
// Returns:
//   - A configured InterceptorFactory that will create ping interceptors
func CreateInterceptorFactory(options ...Option) *InterceptorFactory {
	return &InterceptorFactory{
		opts: options,
	}
}

// NewInterceptor creates and configures a new ping interceptor instance.
// It initializes the base NoOpInterceptor structure, creates a ping manager,
// and applies all stored options to customize the interceptor's behavior.
// This method implements the interceptor.Factory interface.
//
// Parameters:
//   - ctx: Context that controls the lifetime of the interceptor
//   - id: Unique identifier for the interceptor
//
// Returns:
//   - A configured ping interceptor
//   - Any error encountered during interceptor creation or configuration
func (factory *InterceptorFactory) NewInterceptor(ctx context.Context, id string) (interceptor.Interceptor, error) {
	pongInterceptor := &Interceptor{
		NoOpInterceptor: interceptor.NoOpInterceptor{
			ID:  id,
			Ctx: ctx,
		},
		states: make(map[interceptor.Connection]*state),
	}

	for _, option := range factory.opts {
		if err := option(pongInterceptor); err != nil {
			return nil, err
		}
	}

	return pongInterceptor, nil
}
