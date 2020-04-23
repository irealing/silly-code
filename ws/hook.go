package ws

import (
	"log"
	"net/http"
)

type SessionHook interface {
	BeforeAccept(r *http.Request) error
	OnAccept(session *Session, r *http.Request) error
	OnMessage(session *Session, message Message) error
	OnClose(session *Session) error
}

type defaultHook struct {
}

func (*defaultHook) BeforeAccept(r *http.Request) error {
	return nil
}

func (*defaultHook) OnAccept(session *Session, r *http.Request) error {
	wsLogger.Debugf("session %d accept %s", session.id, r.RequestURI)
	return nil
}

func (*defaultHook) OnMessage(session *Session, message Message) error {
	wsLogger.Debugf("recv msg from session %d %s", session.id, message.Bytes())
	log.Println(session.Send(message))
	return nil
}

func (*defaultHook) OnClose(session *Session) error {
	wsLogger.Info("close session ", session.id)
	return nil
}
