package webshell

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"slices"
	"sync/atomic"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

type loggerKey struct{}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func LoggerFrom(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

type Server struct {
	Context      context.Context
	Conn         *websocket.Conn
	WelcomeMessage string

	Token        string
	IsTokenValid atomic.Bool

	WantClose     chan struct{}
	SendToClient  chan []byte
	SendToPty     chan ClientMessage
}

func NewServer(ctx context.Context, conn *websocket.Conn, welcomeMessage string) *Server {
	return &Server{
		Context: ctx,
		Conn: conn,
		WelcomeMessage: welcomeMessage,
		WantClose: make(chan struct{}),
		SendToClient: make(chan []byte, 1024),
		SendToPty: make(chan ClientMessage, 1024),
	}
}

func (s *Server) Close() {
	close(s.SendToPty)
	s.Conn.Close()
}

func (s *Server) Preflight() error {
	go s.ProcessOutboundMessages()
	return s.WaitForAuth()
}

func (s *Server) Start(ptmx *os.File) {
	logger := LoggerFrom(s.Context)

	go s.ProcessPtyInput(ptmx)
	go s.ProcessPtyOutput(ptmx)

	logger.Info("reading messages")
	defer logger.Info("no more messages")
	for {
		if !s.IsTokenValid.Load() {
			err := s.WaitForAuth()
			if err != nil {
				logger.Error("waiting for auth failed", "err", err)
				break
			}
		}

		m, err := s.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				logger.Info("closing connection")
				break
			}
			logger.Error("could not read message", "err", err)
			continue
		}

		s.SendToPty <- m
	}
}

func (s *Server) SetToken(token string) {
	s.Token = token
	s.IsTokenValid.Store(token != "") // FIXME
}

func (s *Server) ReadMessage() (ClientMessage, error) {
	logger := LoggerFrom(s.Context)

	mt, data, err := s.Conn.ReadMessage()
	if err != nil {
		return ClientMessage{}, err
	}

	if mt != websocket.BinaryMessage {
		return ClientMessage{}, errors.New("expected binary message")
	}

	message, err := ParseClientMessage(data)
	logger.Debug("read message", "message", message, "err", err)
	return message, err
}

func (s *Server) WaitForAuth() error {
	logger := LoggerFrom(s.Context)

	queue := make([]ClientMessage, 0, 64)

	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		message := ServerMessageAuthenticate().Encode(&bytes.Buffer{})
		for range ticker.C {
			s.SendToClient <- message
		}
	}()

	for {
		m, err := s.ReadMessage()
		if err != nil {
			return err
		}

		if m.Kind != ClientMessageKindAuthenticate {
			if len(queue) >= 64 {
				return errors.New("client sent too many messages before authenticating")
			}
			queue = append(queue, m)
			continue
		}

		ticker.Stop()
		logger.Debug("authenticated", "token", m.Authenticate.AccessToken)
		s.SetToken(m.Authenticate.AccessToken)
		break
	}

	for _, m := range queue {
		s.SendToPty <- m
	}

	return nil
}

func (s *Server) ProcessOutboundMessages() {
	logger := LoggerFrom(s.Context)

	logger.Info("outbound processing started")
	defer logger.Info("outbound processing closed")

	for {
		select {
		case <-s.WantClose:
			cleanClose := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Shell exited")
			s.Conn.WriteMessage(websocket.CloseMessage, cleanClose)
			return
		case item := <-s.SendToClient:
			err := s.Conn.WriteMessage(websocket.BinaryMessage, item)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err) {
					return
				}
				if errors.Is(err, websocket.ErrCloseSent) {
					return
				}
				if errors.Is(err, net.ErrClosed) {
					return
				}
				logger.Error("could not write to socket", "err", err)
				continue
			}
		}
	}
}

func (s *Server) ProcessPtyOutput(ptmx *os.File) {
	logger := LoggerFrom(s.Context)

	logger.Info("pty output started")
	defer logger.Info("pty output closed")
	defer close(s.SendToClient)

	buffer := bytes.Buffer{}
	buffer.Grow(1024)

	if s.WelcomeMessage != "" {
		s.SendToClient <- slices.Clone(ServerMessageInput([]byte(s.WelcomeMessage)).Encode(&buffer))
	}

	for {
		buffer.Reset()

		var data [1024]byte
		size, err := ptmx.Read(data[:])
		switch {
		case errors.Is(err, io.EOF):
			close(s.WantClose)
			return
		case err != nil:
			logger.Error("could not read from tty", "err", err)
			continue
		}
		s.SendToClient <- slices.Clone(ServerMessageInput(data[:size]).Encode(&buffer))
	}
}

func (s *Server) ProcessPtyInput(ptmx *os.File) {
	logger := LoggerFrom(s.Context)

	logger.Info("pty input started")
	defer logger.Info("pty input closed")

	for message := range s.SendToPty {
		switch message.Kind {
		case ClientMessageKindInvalid: continue
		case ClientMessageKindAuthenticate: continue
		case ClientMessageKindResize:
			logger.Debug("resizing", "size", message.Resize)
			err := pty.Setsize(ptmx, &pty.Winsize{
				Rows: message.Resize.Rows,
				Cols: message.Resize.Cols,
				X: message.Resize.XPixel,
				Y: message.Resize.YPixel,
			})
			if err != nil {
				logger.Error("could not resize pty", "err", err)
			}
			continue
		case ClientMessageKindInput:
			_, err := ptmx.Write(message.Input.Data)
			if err != nil {
				logger.Error("could not write to pty", "err", err)
			}
			continue
		default:
			logger.Error("unhandled condition in switch", "message", message)
			continue
		}
	}
}
