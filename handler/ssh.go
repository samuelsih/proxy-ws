package handler

import (
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

type Handler struct {
	websocketConn *websocket.Conn
	stdout        io.Reader
	stdin         io.WriteCloser
	addr          string
	port          string
	user          string
	password      string
	client        *ssh.Client
	session       *ssh.Session
	Done          chan struct{}
}

func New(conn *websocket.Conn, addr string, port string, user string, password string) (*Handler, error) {
	client := &Handler{websocketConn: conn, addr: addr, port: port, user: user, password: password}

	return client, client.setup()
}

func (h *Handler) setup() error {
	var err error

	defer func(errd error) {
		if errd != nil {
			h.CloseSSHConnection()
		}
	}(err)

	clientConfig := &ssh.ClientConfig{
		User: h.user,
		Auth: []ssh.AuthMethod{
			ssh.Password(h.password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	h.client, err = ssh.Dial("tcp", net.JoinHostPort(h.addr, h.port), clientConfig)
	if err != nil {
		err = fmt.Errorf("Cannot dial ssh: %w", err)
		return err
	}

	return nil
}

func (h *Handler) Prepare() error {
	session, err := h.client.NewSession()
	if err != nil {
		return fmt.Errorf("Cannot create ssh session: %w", err)
	}

	h.session = session

	h.stdout, err = h.session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Cannot acquire ssh stdout pipe: %w", err)
	}

	h.stdin, err = h.session.StdinPipe()
	if err != nil {
		return fmt.Errorf("Cannot acquire ssh stdin pipe: %w", err)
	}

	defer func() {
		if h.stdin == nil {
			return
		}

		if err != nil {
			if errClose := h.stdin.Close(); errClose != nil {
				slog.Error("Cannot close ssh stdin", "err", errClose)
			}
		}
	}()

	terminalModes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	err = h.session.RequestPty("xterm", 60, 40, terminalModes)
	if err != nil {
		return fmt.Errorf("Cannot generate xterm pty %w", err)
	}

	err = h.session.Shell()
	if err != nil {
		return fmt.Errorf("Cannot create shell %w", err)
	}

	return nil
}

func (h *Handler) CloseSSHConnection() {
	if h.client == nil {
		return
	}

	if err := h.client.Close(); err != nil {
		slog.Error("Cannot close ws connection", "err", err)
	}
}
