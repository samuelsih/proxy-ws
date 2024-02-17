package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp"
)

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	_, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
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
	mux.HandleFunc("/ws", websocketHandler)

	engine := nbhttp.NewEngine(nbhttp.Config{
		Network:                 "tcp",
		Addrs:                   []string{addr},
		ReleaseWebsocketPayload: true,
		Handler:                 mux,
	})

	err := engine.Start()
	if err != nil {
		logging.Error("Error starting engine: %v", err)
		return
	}

	idle := make(chan os.Signal, 1)
	signal.Notify(idle, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	<-idle

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	engine.Shutdown(ctx)
}
