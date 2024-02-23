package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

var (
	messageWait = 10 * time.Second
	zeroTime    time.Time
)

func (h *Handler) CaptureStdout() {
	defer func() { h.Done <- struct{}{} }()

	buffer := make([]byte, 512)
	for {
		time.Sleep(10 * time.Millisecond)

		n, readErr := h.stdout.Read(buffer)
		if n > 0 {
			h.websocketConn.SetWriteDeadline(time.Now().Add(messageWait))
			if err := h.websocketConn.WriteMessage(websocket.TextMessage, buffer[:n]); err != nil {
				slog.Error("Cannot write message in websocket", "err", err)
				return
			}
		}

		if readErr != nil {
			slog.Error("Cannot read from stdout", "err", readErr)
		}
	}
}

func (h *Handler) CaptureStdin() {
	defer func() { h.Done <- struct{}{} }()

	h.websocketConn.SetReadDeadline(zeroTime)

	for {
		msgType, connReader, err := h.websocketConn.NextReader()
		if err != nil {
			if websocket.IsCloseError(err) {
				slog.Info("Connection closed from the client")
				return
			}

			slog.Error("Cannot read from websocket", "err", err)
			return
		}

		if msgType != websocket.BinaryMessage {
			if _, err := io.Copy(h.stdin, connReader); err != nil {
				slog.Error("Cannot copy to stdin", "err", err)
				return
			}

			continue
		}

		data := make([]byte, 512)
		n, err := connReader.Read(data)
		if err != nil {
			slog.Error("Cannot read in connection reader", "err", err)
			return
		}

		var term terminal
		if err := json.Unmarshal(data[:n], &term); err != nil {
			slog.Error("Cannot get json data", "err", err)
			return
		}

		if err := h.session.WindowChange(term.Height, term.Width); err != nil {
			slog.Error("Cannot resize ssh session terminal", "err", err)
		}
	}
}
