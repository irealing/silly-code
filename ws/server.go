package ws

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Server interface {
	Accept(w http.ResponseWriter, r *http.Request) error
	Dispose()
	Broadcast(message Message) (int, int, error)
	GetSession(...uint64) []*Session
}

type mtServer struct {
	cancel  context.CancelFunc
	ctx     context.Context
	manager *SessionManager
	hook    SessionHook
	Upgrade websocket.Upgrader
	logger  *slog.Logger
}

func (server *mtServer) GetSession(idx ...uint64) []*Session {
	if len(idx) == 0 {
		return nil
	}
	ss := make([]*Session, len(idx))
	cursor := 0
	for _, v := range idx {
		if s, e := server.manager.GetSession(v); e {
			if !e {
				continue
			}
			ss[cursor] = s
			cursor += 1
		}
	}
	return ss[0:cursor]
}

func (server *mtServer) Broadcast(message Message) (int, int, error) {
	return server.manager.Broadcast(message)
}

func (server *mtServer) Dispose() {
	server.manager.Close()
}

func (server *mtServer) Accept(w http.ResponseWriter, r *http.Request) error {
	if err := server.hook.BeforeAccept(r); err != nil {
		server.logger.Warn("before accept error", "remoteAddr", r.RemoteAddr, "err", err)
		return err
	}
	conn, err := server.Upgrade.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	session := server.manager.NewSession(conn)
	if err := server.hook.OnAccept(session, r); err != nil {
		session.Close()
		return err
	}
	if err := server.manager.Put(session); err != nil {
		session.Close()
		return err
	}
	return nil
}

func NewServer(ctx context.Context, logger *slog.Logger, hook SessionHook, rwCache, retry, ttl int) Server {
	c, cancel := context.WithCancel(ctx)
	if hook == nil {
		hook = &defaultHook{}
	}
	hook = &wrappedHook{hook: hook}
	mg := newManager(ctx, logger, time.Duration(ttl)*time.Second, hook, retry, rwCache)
	return &mtServer{cancel: cancel, ctx: c, manager: mg, hook: hook, logger: logger}
}
