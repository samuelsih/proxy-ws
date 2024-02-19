package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ITS-Nabu/its-nabu-proxy-ws/handler"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  512,
	WriteBufferSize: 512,
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	defer func() {
		if errClose := conn.Close(); errClose != nil {
			slog.Error("Cannot close websocket connection", "err", errClose)
		}
	}()

	sshClient, err := handler.New(conn, "", "", "", "")
	if err != nil {
		slog.Error("Error ssh client", "err", err)
		return
	}

	defer sshClient.CloseSSHConnection()

	err = sshClient.Prepare()
	if err != nil {
		slog.Error("Cannot prepare ssh client", "err", err)
		return
	}

	go sshClient.CaptureStdin()
	go sshClient.CaptureStdout()

	<-sshClient.Done
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
