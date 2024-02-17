package handler

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"

	"golang.org/x/crypto/ssh"
	"nhooyr.io/websocket"
)

type SSHClient struct {
	conn     *websocket.Conn
	addr     string
	port     string
	user     string
	password string
	client   *ssh.Client
	doneChan chan struct{}
}

func New(conn *websocket.Conn, addr string, port string, user string, password string) (*SSHClient, error) {
	client := &SSHClient{conn: conn, addr: addr, port: port, user: user, password: password}

	return client, client.setup()
}

func (c *SSHClient) setup() error {
	var err error

	defer func(errd error) {
		if err != nil {
			c.CloseEverything()
		}
	}(err)

	clientConfig := &ssh.ClientConfig{
		User: c.user,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	c.client, err = ssh.Dial("tcp", net.JoinHostPort(c.addr, c.port), clientConfig)
	if err != nil {
		err = fmt.Errorf("Cannot dial ssh: %w", err)
		return err
	}

	return nil
}

func (c *SSHClient) Run(cmd string) (bytes.Buffer, error) {
	var (
		stdout bytes.Buffer
	)

	session, err := c.client.NewSession()
	if err != nil {
		return stdout, fmt.Errorf("Cannot acquire ssh session: %w", err)
	}

	defer session.Close()

	session.Stdout = &stdout

	err = session.Run(cmd)
	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("Error command: %s --> %w", cmd, err)
	}

	return stdout, nil
}

func (c *SSHClient) CloseEverything() {
	c.closeSSHConnection()
	c.closeWSConnection()
}

func (c *SSHClient) closeWSConnection() {
	if c.conn == nil {
		return
	}

	if err := c.conn.Close(websocket.StatusNormalClosure, "Closed gracefully"); err != nil {
		slog.Error("Cannot close ws connection", "err", err)
	}
}

func (c *SSHClient) closeSSHConnection() {
	if c.client == nil {
		return
	}

	if err := c.client.Close(); err != nil {
		slog.Error("Cannot close ws connection", "err", err)
	}
}
