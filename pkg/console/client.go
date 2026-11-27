// Copyright 2026 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package console

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	// TODO: Checkout https://websocket.org/guides/languages/go/

	"github.com/gorilla/websocket"
)

var (
	DefaultConnectMessage    = []byte(`Connected to server (press enter to continue)`)
	DefaultDisconnectMessage = []byte("\nDisconnected from server\n")
)

type ClientConfig struct {
	BaseURL     string
	AccessToken func() []byte
	Dialer      *websocket.Dialer
}

type client struct {
	conn        *websocket.Conn
	baseURL     string
	dialer      *websocket.Dialer
	AccessToken func() []byte

	vm   string
	user string
}

func (c *client) ResourceName() string {
	return fmt.Sprintf("%s/%s", c.vm, c.user)
}

func (c *client) Conn() *websocket.Conn {
	return c.conn
}

func (c *client) Dial(ctx context.Context, vm, user string) (*client, error) {
	if c.AccessToken == nil {
		return nil, fmt.Errorf("websocket connection: access token is nil, should be func")
	}

	c.vm = vm
	c.user = user

	var err error
	if c.dialer == nil {
		const bufferSize = 10240
		c.dialer = &websocket.Dialer{
			ReadBufferSize:    bufferSize,
			WriteBufferSize:   bufferSize,
			EnableCompression: true,
			HandshakeTimeout:  45 * time.Second,
			Proxy:             websocket.DefaultDialer.Proxy,
		}
	}

	b64Token := base64.URLEncoding.EncodeToString(c.AccessToken())
	c.dialer.Subprotocols = []string{
		"base64url.bearer.authorization.k8s.io." + b64Token,
		"multiplexed.console.dockyards.io",
	}

	websocketURL := fmt.Sprintf("%s/v1/virtualmachines/console", c.baseURL)

	c.conn, _, err = c.dialer.DialContext(ctx, websocketURL, http.Header{})
	if err != nil {
		return nil, err
	}

	return c, err
}

func (c *client) Close() {
	if c.conn == nil {
		return
	}
	c.conn.Close()
}

func (c *client) WriteMessage(messageType int, data []byte) error {
	return c.conn.WriteMessage(messageType, data)
}

func (c *client) WriteMessagesChannel(messages chan []byte) error {
	for message := range messages {
		err := c.WriteMessage(websocket.BinaryMessage, message)
		if err != nil {
			return errors.Join(errors.New("could not write message:"), err)
		}
	}
	return nil
}

func (c *client) ReadMessage() (*ServerMessage, error) {
	return c.readMessage()
}

func (c *client) readMessage() (*ServerMessage, error) {
	_, raw, err := c.conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err) {
			return nil, err
		}
		closedErr, failedWSConn := errors.AsType[*websocket.CloseError](err)
		if failedWSConn {
			return nil, closedErr
		}
		return nil, nil
	}

	if raw == nil {
		return nil, errors.New("connection closed")
	}

	message, err := ParseServerMessage(raw)
	if err != nil {
		return nil, fmt.Errorf("could not parse message: %s", err)
	}

	return &message, nil
}

func (c *client) KeepAlive(ctx context.Context, timeout time.Duration) {
	lastResponse := time.Now()
	c.conn.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	go func() {
		for {
			err := c.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			if err != nil {
				return
			}
			time.Sleep(timeout / 2)
			select {
			case <-ctx.Done():
				return
			default:
				if time.Since(lastResponse) > timeout {
					c.Close()
					return
				}
			}
		}
	}()
}

func NewDialer(cfg ClientConfig) *client {
	strippedURL, isHTTPS := strings.CutPrefix(cfg.BaseURL, "https://")
	strippedURL, _ = strings.CutPrefix(strippedURL, "http://")
	protocol := "ws"
	if isHTTPS {
		protocol = "wss"
	}

	c := &client{
		AccessToken: cfg.AccessToken,
		baseURL:     fmt.Sprintf("%s://%s", protocol, strippedURL),
	}

	return c
}
