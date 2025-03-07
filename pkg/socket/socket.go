package socket

import (
	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type API struct {
	settings            *Settings
	interceptorRegistry *interceptor.Registry
}

type APIOption = func(*API) error

func WithSocketSettings(settings *Settings) APIOption {
	return func(api *API) error {
		api.settings = settings
		return nil
	}
}

func WithInterceptorRegistry(registry *interceptor.Registry) APIOption {
	return func(api *API) error {
		api.interceptorRegistry = registry
		return nil
	}
}

func CreateSocketFactory(options ...APIOption) (*API, error) {
	api := &API{
		settings:            nil,
		interceptorRegistry: nil,
	}

	for _, option := range options {
		if err := option(api); err != nil {
			return nil, err
		}
	}

	if api.settings == nil {
		api.settings = &Settings{}
		if err := RegisterDefaultSettings(api.settings); err != nil {
			return nil, err
		}
	}

	return api, nil
}

func (api *API) CreateWebSocket(id string, options ...Option) (*Socket, error) {
	socket := &Socket{
		id: id,
	}

	interceptors, err := api.interceptorRegistry.Build(id)
	if err != nil {
		return nil, err
	}

	socket.interceptor = interceptors

	for _, option := range options {
		if err := option(socket); err != nil {
			return nil, err
		}
	}

	return socket, nil
}

type Socket struct {
	id          string
	connections map[string]*websocket.Conn
	interceptor interceptor.Interceptor
}

func (socket *Socket) Serve() {

}
