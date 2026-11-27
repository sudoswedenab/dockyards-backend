package console_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/sudoswedenab/dockyards-backend/pkg/console"
)

var upgrader = websocket.Upgrader{}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	var i int
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		err = c.WriteMessage(mt, fmt.Appendf(nil, "%s[%d]", message, i))
		if err != nil {
			break
		}
		i++
	}
}

func TestWebsocketClient(t *testing.T) {
	ctx := t.Context()
	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(echo))
	defer s.Close()

	c, err := console.NewDialer(console.ClientConfig{
		BaseURL: s.URL,
		AccessToken: func() []byte {
			return []byte("access-token")
		},
	}).Dial(ctx, "sudosweden", "dockyards-xyz")
	require.NoError(t, err)
	defer c.Close()

	// Start the command with a pty.
	ptmx, tty, err := pty.Open()
	require.NoError(t, err)

	defer func() { _ = ptmx.Close() }() // Best effort
	defer func() { _ = tty.Close() }()  // Best effort

	var messages = make(chan []byte, 1024)

	go func(required require.Assertions) {
		err := c.WriteMessagesChannel(messages)
		required.NoError(err, "WriteMessage")
	}(*require.New(t))

	b := []byte{}
	buffer := bytes.NewBuffer(b)

	// Send message to server, read response and check to see if it's what we expect.
	for i := range 10 {
		required := require.New(t)

		formatMessage(messages, buffer, []byte("hello"))

		p, err := c.ReadMessage()
		required.NoError(err, "ReadMessage[%d]", i)

		expected := fmt.Sprintf("%s[%d]", "hello", i)
		required.Equal(expected, string(p.Input.Data))
	}
}

func formatMessage(messages chan []byte, buffer *bytes.Buffer, msg []byte) error {
	buffer.Reset()

	messages <- console.ClientMessageInput(1, msg).Encode(buffer)
	return nil
}
