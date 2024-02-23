package main

import (
	"context"
	"encoding/json"
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
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type credential struct {
	IP       string `json:"ip"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Cannot upgrade to websocket", "err", err)
		w.Write([]byte(err.Error()))
		return
	}

	var cred credential

	defer func() {
		if errClose := conn.Close(); errClose != nil {
			slog.Error("Cannot close websocket connection", "err", errClose)
		}
	}()

	_, reader, err := conn.NextReader()
	if err != nil {
		slog.Error("Cannot read from websocket", "err", err)
		return
	}

	if err := json.NewDecoder(reader).Decode(&cred); err != nil {
		slog.Error("Cannot parse credentials", "err", err, "username", cred.Username, "password", cred.Password, "ip", cred.IP)
		conn.WriteMessage(websocket.BinaryMessage, []byte("Cannot parse credentials"))
		return
	}

	sshClient, err := handler.New(conn, cred.IP, "22", cred.Username, cred.Password)
	if err != nil {
		slog.Error("Error ssh client", "err", err)
		return
	}

	defer sshClient.CloseSSHConnection()
	defer close(sshClient.Done)

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
	defer close(idle)

	if err := server.Shutdown(ctx); err != nil {
		slog.ErrorContext(ctx, "Error shutting down websocket server", "err", err.Error())
	}

	slog.Info("Server shutdown completely")
}
