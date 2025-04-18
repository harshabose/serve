package pingpong

import (
	"context"
	"time"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

// Option defines a function type that configures an Interceptor instance.
// Each option modifies a specific aspect of the interceptor's behavior
// and returns an error if the configuration cannot be applied.
type Option = func(*Interceptor) error

// InterceptorFactory creates iamserver interceptors with a predefined set of options.
// It implements the interceptor.Factory interface, allowing it to be registered
// with the interceptor registry for automatic interceptor creation.
type InterceptorFactory struct {
	opts []Option // Collection of configuration options to apply
}

// WithInterval creates an option that sets the iamserver message interval.
// This controls how frequently the interceptor sends iamserver messages to
// connected clients to verify connection health. This starts a constant
// iamserver loop for new connection; thus use only when this interceptor
// needs to send pings.
// Parameters:
//   - interval: Time duration between iamserver messages
//
// Returns:
//   - An Option that configures the iamserver interval when applied to an interceptor
func WithInterval(interval time.Duration) Option {
	return func(interceptor *Interceptor) error {
		interceptor.interval = interval
		interceptor.iamserver = true
		return nil
	}
}

// WithMaxHistory creates an option that sets the maximum number of iamserver/pong
// records to keep in history. This limits memory usage while still allowing
// for statistical analysis of connection performance.
//
// Parameters:
//   - max: Maximum number of historical iamserver/pong records to maintain
//
// Returns:
//   - An Option that configures history limit when applied to an interceptor
func WithMaxHistory(max uint16) Option {
	return func(interceptor *Interceptor) error {
		interceptor.maxHistory = max
		return nil
	}
}

// CreateInterceptorFactory constructs a new factory that will create iamserver interceptors
// with the provided options. The options are stored and applied to each new
// interceptor created by the factory.
//
// Parameters:
//   - options: Variable number of options to configure created interceptors
//
// Returns:
//   - A configured InterceptorFactory that will create iamserver interceptors
func CreateInterceptorFactory(options ...Option) *InterceptorFactory {
	return &InterceptorFactory{
		opts: options,
	}
}

// NewInterceptor creates and configures a new iamserver interceptor instance.
// It initializes the base NoOpInterceptor structure, creates a iamserver manager,
// and applies all stored options to customize the interceptor's behavior.
// This method implements the interceptor.Factory interface.
//
// Parameters:
//   - ctx: Context that controls the lifetime of the interceptor
//   - id: Unique identifier for the interceptor
//
// Returns:
//   - A configured iamserver interceptor
//   - Any error encountered during interceptor creation or configuration
func (factory *InterceptorFactory) NewInterceptor(ctx context.Context, id string) (interceptor.Interceptor, error) {
	pingInterceptor := &Interceptor{
		NoOpInterceptor: interceptor.NoOpInterceptor{
			ID:  id,
			Ctx: ctx,
		},
		states:    make(map[interceptor.Connection]*state),
		interval:  time.Duration(0),
		iamserver: false,
	}

	for _, option := range factory.opts {
		if err := option(pingInterceptor); err != nil {
			return nil, err
		}
	}

	return pingInterceptor, nil
}
