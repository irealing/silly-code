package ws

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
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

type wrappedHook struct {
	hook SessionHook
}

func (w *wrappedHook) handlePanic(event string) {
	err := recover()
	if err == nil {
		return
	}
	buf := make([]byte, 2048)
	n := runtime.Stack(buf, false)
	fmt.Printf("%s panic %s \n %s", event, err, string(buf[:n]))
}
func (w *wrappedHook) BeforeAccept(r *http.Request) error {
	defer w.handlePanic("BeforeAccept")
	return w.hook.BeforeAccept(r)
}

func (w *wrappedHook) OnAccept(session *Session, r *http.Request) error {
	defer w.handlePanic("OnAccept")
	return w.hook.OnAccept(session, r)
}

func (w *wrappedHook) OnMessage(session *Session, message Message) error {
	defer w.handlePanic("OnMessage")
	return w.hook.OnMessage(session, message)
}

func (w *wrappedHook) OnClose(session *Session) error {
	defer w.handlePanic("OnClose")
	return w.hook.OnClose(session)
}
