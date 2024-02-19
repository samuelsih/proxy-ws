package handler

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gorilla/websocket"
)

type terminal struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (h *Handler) getTerminalSize() (terminal, error) {
	messageType, msg, err := h.websocketConn.ReadMessage()
	if err != nil {
		return terminal{}, fmt.Errorf("Can't read message: %w", err)
	}

	if messageType != websocket.BinaryMessage {
		return terminal{}, errors.New("Invalid message type")
	}

	var term terminal
	err = json.Unmarshal(msg, &term)
	if err != nil {
		return terminal{}, fmt.Errorf("Can't parse message: %w'", err)
	}

	return term, nil
}
