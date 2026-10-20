// Copyright 2026 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package console

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sudoswedenab/dockyards-backend/api/jwt"
)

type EventKind int
const (
	EventKindAuthorizationChange EventKind = iota
	EventKindInput
)

type AuthorizationChange struct {
	IsAllowed    bool
}

type Event struct {
	AuthorizationChange AuthorizationChange
	Input               []byte

	Stream *Stream
	Kind   EventKind
}

func AuthorizationChangeEvent(stream *Stream, isAllowed bool) Event {
	return Event{
		AuthorizationChange: AuthorizationChange{
			IsAllowed: isAllowed,
		},
		Stream: stream,
		Kind: EventKindAuthorizationChange,
	}
}

func InputEvent(stream *Stream, input []byte) Event {
	return Event{
		Input: input,
		Stream: stream,
		Kind: EventKindInput,
	}
}

type RequestHandlerOptions struct {
	Events     chan Event
	AccessKey  *ecdsa.PublicKey
	Upgrader   *websocket.Upgrader
	IsAllowed  func(ctx context.Context, server *Server, stream *Stream) bool
	Start      func(ctx context.Context, server *Server, stream *Stream) error
}

func HandleRequest(w http.ResponseWriter, r *http.Request, options RequestHandlerOptions) {
	logger := LoggerFrom(r.Context())
	subject, err := jwt.SubjectFromRequest(r, options.AccessKey)
	if err != nil {
		logger.Error("could not get subject from request", "err", err)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}
	logger.Info("opening console", "subject", subject)

	conn, err := options.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error("could not upgrade connection", "err", err)

		return
	}

	console := NewServer(ServerOptions{
		Subject: subject,
		Context: r.Context(),
		Conn: conn,
		Events: options.Events,
		IsAllowed: options.IsAllowed,
		Start: options.Start,
	})
	defer console.Close()

	err = console.Run()
	if err != nil {
		logger.Error("could not upgrade connection", "err", err)

		return
	}
}

type Stream struct {
	Cancel func()
	Done func() <-chan struct{}

	ID uint16
	Input chan ClientMessage
	ResourceName string

	server *Server
}

func (c *Stream) Close() {
	if c == nil {
		return
	}
	c.server.sendToClient <- ServerMessageDisconnect(c.ID).Encode(&bytes.Buffer{})
	c.Cancel()
	c.server.mtx.Lock()
	delete(c.server.streams, c.ID)
	c.server.mtx.Unlock()
}

type ServerOptions struct {
	Subject      string
	Context      context.Context
	Conn         *websocket.Conn
	Events       chan Event

	IsAllowed        func(ctx context.Context, server *Server, stream *Stream) bool
	Start            func(ctx context.Context, server *Server, stream *Stream) error
}

type Server struct {
	ServerOptions

	mtx     sync.Mutex
	streams map[uint16]*Stream

	sendToClient  chan []byte
	cancel func()
}

func NewServer(options ServerOptions) *Server {
	ctx, cancel := context.WithCancel(options.Context)
	options.Context = ctx
	return &Server{
		ServerOptions: options,

		sendToClient: make(chan []byte, 1024),
		cancel: cancel,
	}
}

func (s *Server) Close() {
	s.mtx.Lock()
	for _, c := range s.streams {
		c.Close()
	}
	s.mtx.Unlock()
	s.cancel()
}

func (s *Server) Run() error {
	logger := LoggerFrom(s.Context)

	go s.ProcessOutboundMessages()
	logger.Info("reading messages")
	defer logger.Info("no more messages")
	for {
		m, err := s.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				logger.Info("closing connection", "err", err)
				break
			}
			logger.Error("could not read message", "err", err)
			continue
		}

		err = s.handleMessage(m)
		if err != nil {
			logger.Error("could not handle message", "err", err)
			continue
		}
	}

	return nil
}

func (s *Server) handleMessage(m ClientMessage) error {
	switch m.Kind {
	case ClientMessageKindInvalid:
		return nil
	case ClientMessageKindConnect:
		return s.connect(m.Connect)
	case ClientMessageKindDisconnect:
		return s.disconnect(m.Disconnect)
	case ClientMessageKindInput:
		return s.input(m.Input)
	case ClientMessageKindResize:
		return s.resize(m.Resize)
	default:
		return fmt.Errorf("unknown message kind: %v", m.Kind)
	}
}

func (s *Server) connect(m ClientConnect) error {
	s.mtx.Lock()
	if s.streams == nil {
		s.streams = make(map[uint16]*Stream)
	}

	stream := s.streams[m.StreamID]
	if stream != nil {
		s.mtx.Unlock()
		if stream.ResourceName == m.ResourceName {
			return nil
		}
		s.sendToClient <- ServerMessageDisconnect(m.StreamID).Encode(&bytes.Buffer{})
		return errors.New("client tried to connect with an id that is already in use")
	}

	// This may take some arbitrary amount of time, so let's not hold the lock during
	// that period.
	s.mtx.Unlock()
	ctx := context.Background()

	stream = &Stream{
		ID: m.StreamID,
		Input: make(chan ClientMessage, 1024),
		ResourceName: m.ResourceName,
		server: s,
	}
	err := s.Start(ctx, s, stream)
	if err != nil {
		s.sendToClient <- ServerMessageDisconnect(m.StreamID).Encode(&bytes.Buffer{})
		return fmt.Errorf("could not start connection: %w", err)
	}
	s.sendToClient <- ServerMessageConnect(m.StreamID).Encode(&bytes.Buffer{})
	s.mtx.Lock()
	s.streams[m.StreamID] = stream
	s.mtx.Unlock()

	go func() {
		<-stream.Done()
		stream.Close()
	}()

	return nil
}

func (s *Server) disconnect(m ClientDisconnect) error {
	s.mtx.Lock()
	stream := s.streams[m.StreamID]
	s.mtx.Unlock()
	if stream == nil {
		return errors.New("client tried to disconnect from a disconnected stream")
	}

	stream.Close()
	return nil
}

func (s *Server) input(m ClientInput) error {
	s.mtx.Lock()
	stream := s.streams[m.StreamID]
	s.mtx.Unlock()
	if stream == nil {
		return errors.New("client sent input to disconnected stream")
	}

	if !s.IsAllowed(s.Context, s, stream) {
		stream.Close()
		return errors.New("client is not allowed to access resource")
	}

	stream.Input <- ClientMessageInput(m.StreamID, m.Data)

	return nil
}

func (s *Server) resize(m ClientResize) error {
	s.mtx.Lock()
	stream := s.streams[m.StreamID]
	s.mtx.Unlock()
	if stream == nil {
		return errors.New("client sent resize to disconnected resource")
	}

	if !s.IsAllowed(s.Context, s, stream) {
		stream.Close()
		return errors.New("client is not allowed to access resource")
	}

	stream.Input <- ClientMessageResize(m.StreamID, m.Rows, m.Cols, m.XPixel, m.YPixel)

	return nil
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

	message, err := DecodeClientMessage(data)
	logger.Debug("read message", "message", message, "err", err)
	return message, err
}

func (s *Server) ProcessOutboundMessages() {
	logger := LoggerFrom(s.Context)

	logger.Info("outbound processing started")
	defer logger.Info("outbound processing closed")

	for {
		select {
		case <-s.Context.Done():
			cleanClose := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Shell exited")
			_ = s.Conn.WriteMessage(websocket.CloseMessage, cleanClose)
			return
		case event := <-s.Events:
			s.processEvent(event)
		case item := <-s.sendToClient:
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

func (s *Server) processEvent(e Event) {
	logger := LoggerFrom(s.Context)

	switch e.Kind {
	case EventKindAuthorizationChange:
		if !e.AuthorizationChange.IsAllowed {
			logger.Info("subject is no longer allowed resource", "subject", s.Subject, "resourceName", e.Stream.ResourceName)
			e.Stream.Close()

			return
		}
	case EventKindInput:
		if !s.IsAllowed(s.Context, s, e.Stream) {
			return
		}
		data := e.Input
		s.sendToClient <- ServerMessageInput(e.Stream.ID, data).Encode(&bytes.Buffer{})
	}
}

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
