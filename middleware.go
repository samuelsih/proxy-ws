package main

import (
	"log/slog"
	"net/http"
)

func serverLogger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("User Hit", "addr", r.RemoteAddr)
		next(w, r)
	}
}

func serverRecoverer(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				slog.Error("Error panic", "error", err)

				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("There was an internal server error"))
			}

		}()
		next(w, r)
	})
}
