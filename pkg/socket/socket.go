package socket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type API struct {
	settings            *apiSettings
	interceptorRegistry *interceptor.Registry
}

type APIOption = func(*API) error

func WithInterceptorRegistry(registry *interceptor.Registry) APIOption {
	return func(api *API) error {
		api.interceptorRegistry = registry
		return nil
	}
}

func CreateAPI(options ...APIOption) (*API, error) {
	api := &API{
		settings:            &apiSettings{},
		interceptorRegistry: nil,
	}

	if err := registerDefaultAPISettings(api.settings); err != nil {
		return nil, err
	}

	for _, option := range options {
		if err := option(api); err != nil {
			return nil, err
		}
	}

	return api, nil
}

func (api *API) CreateWebSocket(ctx context.Context, id string, options ...Option) (*Socket, error) {
	socket := &Socket{
		id:                  id,
		settings:            &settings{},
		socketAcceptOptions: &websocket.AcceptOptions{},
		ctx:                 ctx,
	}

	interceptors, err := api.interceptorRegistry.Build(ctx, id)
	if err != nil {
		return nil, err
	}

	socket.interceptor = interceptors

	if err := registerDefaultSettings(socket.settings); err != nil {
		return nil, err
	}

	for _, option := range options {
		if err := option(socket); err != nil {
			return nil, err
		}
	}

	return socket.setup(), nil
}

type Socket struct {
	id                  string
	settings            *settings
	server              *http.Server
	router              *http.ServeMux
	handlerFunc         *http.HandlerFunc
	socketAcceptOptions *websocket.AcceptOptions
	interceptor         interceptor.Interceptor
	mux                 sync.RWMutex
	ctx                 context.Context
}

func (socket *Socket) setup() *Socket {
	socket.router = http.NewServeMux()
	socket.server = &http.Server{}
	socket.handlerFunc = socket.baseHandler

	socket.settings.apply(socket)

	return socket
}

func (socket *Socket) serve() error {
	defer socket.close()

	for {
		select {
		case <-socket.ctx.Done():
			return nil
		default:
			if err := socket.server.ListenAndServeTLS(socket.settings.TLSCertFile, socket.settings.TLSKeyFile); err != nil {
				fmt.Println(errors.New("error while serving HTTP server"))
				fmt.Println("trying again...")
			}
		}
	}
}

func (socket *Socket) baseHandler(w http.ResponseWriter, r *http.Request) {
	connection, err := websocket.Accept(w, r, socket.socketAcceptOptions)
	if err != nil {
		fmt.Println(errors.New("error while accepting socket connection"))
	}

	if _, _, err := socket.interceptor.BindSocketConnection(connection, socket, socket); err != nil {
		fmt.Println("error while handling client:", err.Error())
		return
	}

	// READ MESSAGE LOOP HERE

	if err := socket.interceptor.Init(connection); err != nil {
		return
	}
}

func (socket *Socket) close() {

}

func (socket *Socket) Write(connection interceptor.Connection, messageType websocket.MessageType, message message.Message) error {
	ctx, cancel := context.WithTimeout(socket.ctx, 100*time.Millisecond)
	defer cancel()

	data, err := message.Marshal()
	if err != nil {
		return err
	}

	return connection.Write(ctx, messageType, data)
}

func (socket *Socket) Read(connection interceptor.Connection) (websocket.MessageType, message.Message, error) {
	ctx, cancel := context.WithTimeout(socket.ctx, 100*time.Millisecond)
	defer cancel()

	messageType, data, err := connection.Read(ctx)
	if err != nil {
		return websocket.MessageText, nil, err
	}

	msg := &message.BaseMessage{}
	if err := msg.Unmarshal(data); err != nil {
		return websocket.MessageText, nil, err
	}

	return messageType, msg, nil
}
