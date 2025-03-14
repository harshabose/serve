package socket

import "github.com/coder/websocket"

type clients = map[string]*client

type client struct {
	id         string
	connection *websocket.Conn
}

func createClient(connection *websocket.Conn) *client {
	return &client{
		connection: connection,
	}
}
