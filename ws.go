package main

import (
	"unsafe"

	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
)

var (
	upgrader = websocketUpgrader()
)

func websocketUpgrader() *websocket.Upgrader {
	upgr := websocket.NewUpgrader()

	upgr.OnOpen(func(c *websocket.Conn) {
		logging.Info("Websocket opened!, addr %s", c.RemoteAddr().String())
	})

	upgr.OnMessage(func(c *websocket.Conn, messageType websocket.MessageType, data []byte) {
		logging.Info("Websocket message: type: %v, data: %s", messageType, data)
		c.WriteMessage(messageType, data)
	})

	upgr.OnClose(func(c *websocket.Conn, err error) {
		logging.Info("Websocket closed! addr: %s", c.RemoteAddr().String())
	})

	return upgr
}

// Zero alloc bytes to string
func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
