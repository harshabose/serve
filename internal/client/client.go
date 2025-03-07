package client

import "github.com/coder/websocket"

type Client struct {
	id         string
	connection *websocket.Conn
}
