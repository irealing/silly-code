package ws

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type SessionManager struct {
	ctx       context.Context
	cancel    context.CancelFunc
	TTL       time.Duration
	sessions  map[uint64]*Session
	wg        sync.WaitGroup
	rwLocker  sync.RWMutex
	hook      SessionHook
	retry     int
	rwCache   int
	isClosed  int32
	closeOnce sync.Once
	logger    *slog.Logger
}

func newManager(ctx context.Context, logger *slog.Logger, ttl time.Duration, hook SessionHook, retry, rwCache int) *SessionManager {
	c, cancel := context.WithCancel(ctx)
	sessions := make(map[uint64]*Session)
	return &SessionManager{
		ctx: c, cancel: cancel, TTL: ttl, hook: hook, retry: retry, rwCache: rwCache, sessions: sessions,
		logger: logger,
	}
}
func (manager *SessionManager) Context() context.Context {
	return manager.ctx
}
func (manager *SessionManager) NewSession(conn *websocket.Conn) *Session {
	session := newSession(conn, manager, manager.retry, manager.rwCache, manager.logger)
	go manager.startSession(session)
	return session
}
func (manager *SessionManager) startSession(session *Session) {
	defer func(s *Session) {
		if err := recover(); err != nil {
			manager.logger.Debug("session run panic", "session", s.ID(), "error", err)
		}
	}(session)
	err := session.Run()
	if err != nil {
		manager.logger.Info("session run error", "session", session.ID(), "error", err)
	}
}
func (manager *SessionManager) Put(session *Session) error {
	manager.rwLocker.Lock()
	defer manager.rwLocker.Unlock()
	if _, ok := manager.sessions[session.ID()]; ok {
		return errors.New("session already exists")
	}
	manager.sessions[session.ID()] = session
	manager.wg.Add(1)
	manager.logger.Debug("add session ", "session", session.id)
	return nil
}
func (manager *SessionManager) Closed() bool {
	return atomic.LoadInt32(&manager.isClosed) > 0
}
func (manager *SessionManager) GetSession(id uint64) (*Session, bool) {
	manager.rwLocker.RLock()
	defer manager.rwLocker.RUnlock()
	session, ok := manager.sessions[id]
	return session, ok
}
func (manager *SessionManager) Close() {
	manager.closeOnce.Do(manager.close)
}

func (manager *SessionManager) close() {
	manager.cancel()
	manager.wg.Wait()
}
func (manager *SessionManager) Remove(id uint64) {
	manager.rwLocker.Lock()
	defer manager.rwLocker.Unlock()
	_, ok := manager.sessions[id]
	if !ok {
		return
	}
	delete(manager.sessions, id)
	manager.logger.Debug("remove session", "id", id)
	manager.wg.Done()
}
func (manager *SessionManager) Clients() int {
	manager.rwLocker.RLock()
	defer manager.rwLocker.Unlock()
	return len(manager.sessions)
}
func (manager *SessionManager) Broadcast(m Message) (success int, failed int, err error) {
	manager.logger.Info("publish Broadcast ")
	manager.rwLocker.RLock()
	defer manager.rwLocker.RUnlock()
	if manager.isClosed > 0 {
		err = closedSessionManager
		return
	}

	for _, v := range manager.sessions {
		manager.logger.Debug("send message to session", "id", v.id)
		err := v.Send(m)
		if err == nil {
			success += 1
		} else {
			failed += 1
		}
	}
	return
}
