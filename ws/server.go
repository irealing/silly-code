package ws

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/op/go-logging"
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
	logger  *logging.Logger
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
		server.logger.Warning("before accept error", r.RemoteAddr, err)
		server.writeError(w, err)
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
func (server *mtServer) writeError(w http.ResponseWriter, err error) {
	_, err = w.Write([]byte(err.Error()))
	if err != nil {
		server.logger.Warningf("write error %s", err)
	}
}
func NewServer(ctx context.Context, hook SessionHook, rwCache, retry, ttl int) Server {
	c, cancel := context.WithCancel(ctx)
	if hook == nil {
		hook = &defaultHook{}
	}
	hook = &wrappedHook{hook: hook}
	mg := newManager(ctx, time.Duration(ttl)*time.Second, hook, retry, rwCache)
	return &mtServer{cancel: cancel, ctx: c, manager: mg, hook: hook, logger: logging.MustGetLogger("MTubeServer")}
}
