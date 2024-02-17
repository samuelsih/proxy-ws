package sshtool

import (
	"io"
	"os"

	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"golang.org/x/crypto/ssh"
)

type Client struct {
	conn     *websocket.Conn
	addr     string
	user     string
	password string
	client   *ssh.Client
	session  *ssh.Session
	in       io.WriteCloser
	out      io.Reader
	doneChan chan struct{}
}

func New(conn *websocket.Conn, addr string, user string, password string) *Client {
	return &Client{conn: conn, addr: addr, user: user, password: password}
}

func (c *Client) Run() {
	var err error

	defer c.closeWSConnection()

	clientConfig := &ssh.ClientConfig{
		User:            c.user,
		Auth:            []ssh.AuthMethod{
			ssh.Password(c.password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	c.client, err = ssh.Dial("tcp", c.addr, clientConfig)
	if err != nil {
		logging.Error("Cannot dial ssh: %v", err)
		return
	}

	defer c.closeSSHConnection()

	c.session, err = c.client.NewSession()
	if err != nil {
		logging.Error("Cannot acquire ssh session: %v", err)
		return
	}

	defer c.closeSSHSessionConnection()

	c.session.Stderr = os.Stderr
	c.out, err = c.session.StdoutPipe()
	if err != nil {
		logging.Error("Cannot acquire ssh stdout: %v", err)
		return
	}

	c.in, err = c.session.StdinPipe()
	if err != nil {
		logging.Error("Cannot acquire ssh stdin: %v", err)
		return
	}

	defer c.closeIOInputStream()

	// TODO: handle input from user
}

func (c *Client) closeWSConnection() {
	if err := c.conn.Close(); err != nil {
		logging.Error("Cannot close ws connection: %v", err)
	}
}

func (c *Client) closeSSHConnection() {
	if err := c.client.Close(); err != nil {
		logging.Error("Cannot close ws connection: %v", err)
	}
}

func (c *Client) closeSSHSessionConnection() {
	if err := c.session.Close(); err != nil {
		logging.Error("Cannot close ws connection: %v", err)
	}
}

func (c *Client) closeIOInputStream() {
	if err := c.in.Close(); err != nil {
		logging.Error("Cannot close ssh stdin connection: %v", err)
	}
}