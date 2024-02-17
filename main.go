package main

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ITS-Nabu/its-nabu-proxy-ws/handler"
	"nhooyr.io/websocket"
)

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	sshClient, err := handler.New(conn, "localhost", "22", "dev", "password")
	if err != nil {
		slog.Error("Error ssh client", "err", err)
		conn.Write(r.Context(), websocket.MessageType(websocket.StatusInternalError), []byte("Error ssh client: "+err.Error()))
		return
	}

	defer sshClient.CloseEverything()

	for {
		msgType, msg, err := conn.Read(r.Context())
		if err != nil {
			slog.Error("Error read message", "err", err)
			return
		}

		writer, err := conn.Writer(r.Context(), msgType)
		if err != nil {
			slog.Error("Error acquiring writer", "err", err)
			return
		}

		output, err := sshClient.Run(string(msg))
		if err != nil {
			slog.Error("Error getting output", "err", err)
			return
		}

		io.Copy(writer, &output)

		writer.Close()
	}
}

func main() {
	var (
		host string
		port string
	)

	flag.StringVar(&host, "host", "localhost", "host to connect to")
	flag.StringVar(&port, "port", "17542", "Port to listen on")
	flag.Parse()

	addr := net.JoinHostPort(host, port)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", serverRecoverer(websocketHandler))

	server := &http.Server{Handler: mux, Addr: addr}

	slog.Info("Starting server", "addr", addr)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			slog.Error("Error starting websocket server", "err", err.Error())
			os.Exit(1)
		}
	}()

	idle := make(chan os.Signal, 1)
	signal.Notify(idle, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	<-idle

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.ErrorContext(ctx, "Error shutting down websocket server", "err", err.Error())
	}

	slog.Info("Server shutdown completely")
}
