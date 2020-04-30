package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/op/go-logging"
)

var globalSessionID uint64 = 0

type Session struct {
	id        uint64
	conn      *websocket.Conn
	TTL       time.Duration
	manager   *SessionManager
	ctx       context.Context
	cancel    context.CancelFunc
	send      chan Message
	recv      chan Message
	Retry     int
	closeOnce sync.Once
	rw        sync.RWMutex
	hook      SessionHook
	closed    int32
	logger    *logging.Logger
}

func (s *Session) onMessage(message Message) {
	if message.Type() == websocket.CloseMessage {
		s.Close()
	}
	if s.hook == nil {
		return
	}
	if err := s.hook.OnMessage(s, message); err != nil {
		s.logger.Warningf("handled message error %s", err.Error())
	}
}
func (s *Session) onClose() {
	if s.hook == nil {
		return
	}
	if err := s.hook.OnClose(s); err != nil {
		s.logger.Warningf("session %d onClose error %s", s.id, err)
	}
}
func newSession(conn *websocket.Conn, manager *SessionManager, retry int, rwCache int) *Session {
	id := atomic.AddUint64(&globalSessionID, 1)
	ctx, cancel := context.WithCancel(manager.ctx)
	send := make(chan Message, rwCache)
	recv := make(chan Message, rwCache)
	return &Session{
		id: id, conn: conn, TTL: manager.TTL, ctx: ctx,
		cancel: cancel, Retry: retry, send: send, recv: recv, manager: manager, hook: manager.hook,
		logger: wsLogger,
	}
}

func (s *Session) ID() uint64 {
	return s.id
}
func (s *Session) Addr() net.Addr {
	return s.conn.RemoteAddr()
}

func (s *Session) Read() (Message, error) {
	select {
	case msg, ok := <-s.recv:
		if !ok {
			return nil, errors.New("closed session")
		}
		return msg, nil
	case <-time.After(s.TTL):
		return nil, newError(true, false, "read timeout")
	}
}
func (s *Session) ReadJSON(v interface{}) error {
	m, err := s.Read()
	if err != nil {
		return err
	}
	return json.Unmarshal(m.Bytes(), v)
}

func (s *Session) write(m Message) error {
	if err := s.conn.SetWriteDeadline(time.Now().Add(s.TTL)); err != nil {
		return err
	}
	return s.conn.WriteMessage(m.Type(), m.Bytes())
}
func (s *Session) ping() error {
	return s.writeWithRetry(NewRawMessage(websocket.PingMessage, nil))
}
func (s *Session) writeWithRetry(m Message) error {
	var err error
	for i := 0; i < s.Retry+1; i++ {
		err = s.write(m)
		if err != nil && os.IsTimeout(err) {
			continue
		} else {
			return err
		}
	}
	return err
}

func (s *Session) Recv() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			m, err := s.receive()
			if err != nil {
				s.Close()
				continue
			}
			s.putMsg(m)
		}
	}
}

func (s *Session) putMsg(m Message) {
	s.rw.RLock()
	defer s.rw.RUnlock()
	select {
	case <-s.ctx.Done():
		return
	default:
		s.recv <- m
	}
}

func (s *Session) receive() (Message, error) {
	t, data, err := s.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	return &RawMessage{t, data}, nil
}

func (s *Session) Send(m Message) error {
	s.rw.RLock()
	defer s.rw.RUnlock()
	if s.closed > 0 {
		return closedSession
	}
	select {
	case <-s.ctx.Done():
		return closedSession
	case s.send <- m:
		s.logger.Debug("write message in send channel ", s.id)
	}
	return nil
}
func (s *Session) Run() error {
	s.logger.Debug("run session ", s.id)
	t := time.NewTicker(s.TTL)
	defer t.Stop()
	defer s.Close()
	go s.Recv()
loop:
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Debugf("context done , session %d break loop", s.id)
			break loop
		case msg, ok := <-s.send:
			if !ok {
				s.logger.Debugf("session %d send chan closed,break loop", s.id)
				break loop
			}
			err := s.writeWithRetry(msg)
			if err != nil {
				s.logger.Warningf("session %d break loop with error %s ", s.id, err)
				break
			}
		case msg, ok := <-s.recv:
			if !ok {
				break loop
			}
			s.onMessage(msg)
		case <-t.C:
			if err := s.ping(); err != nil {
				s.logger.Debug("ping client ", s.id)
				break loop
			}
		}
	}

	return nil
}
func (s *Session) Closed() bool {
	return atomic.LoadInt32(&s.closed) > 0
}
func (s *Session) Close() {
	s.logger.Debugf("session %d Close was called", s.id)
	s.closeOnce.Do(s.close)
}
func (s *Session) close() {
	s.cancel()
	s.rw.Lock()
	defer s.rw.Unlock()
	atomic.StoreInt32(&s.closed, 1)
	if err := s.conn.Close(); err != nil {
		s.logger.Warningf("session %d connection close error %s", s.id, err.Error())
	}
	s.onClose()
	close(s.send)
	close(s.recv)
	s.manager.Remove(s.id)
}
