package ws

import (
	"log/slog"
	"net/http"
	"runtime"
)

type SessionHook interface {
	// BeforeAccept
	//
	// Call before accept a websocket connection, validate
	BeforeAccept(r *http.Request) error
	// OnAccept
	//
	// call after a websocket a websocket connection established
	OnAccept(session *Session, r *http.Request) error
	// OnMessage
	//
	// callback when receive message
	OnMessage(session *Session, message Message) error
	// OnClose
	//
	// clean session storage here
	OnClose(session *Session) error
}

type defaultHook struct {
}

func (*defaultHook) BeforeAccept(r *http.Request) error {
	return nil
}

func (*defaultHook) OnAccept(session *Session, r *http.Request) error {
	slog.Debug("session accept", "id", session.id, "uri", r.RequestURI, "clientIP", r.RemoteAddr)
	return nil
}

func (*defaultHook) OnMessage(session *Session, message Message) error {
	slog.Debug("recv msg from session", "session", session.id, "data", message.Bytes())
	if err := session.Send(message); err != nil {
		slog.Debug("send message to session error", "session", session.id, "error", err)
	} else {
		return err
	}
	return nil
}

func (*defaultHook) OnClose(session *Session) error {
	slog.Info("close session ", "session", session.id)
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
	slog.Error("panic", "event", "event", "err", err, "stack", string(buf[:n]))
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
