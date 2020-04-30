## Silly-Code WS![goproxy.cn](goproxy.cn/stats/github.com/irealing/silly-code/ws/badges/download-count.svg)

  简单 WebSocket服务脚手架工具。
  以会话`ws.Session`为核心，每一个WS连接作为一个会话，实现自定义`ws.SessionHook`即可维护会话的各个声明周期。
 ###会话生命周期
 * BeforeAccept
    建立连接前调用，即服务器端收到客户端连接请求时调用
 * Accept
    创建连接后调用(ws.Session对象创建后)
 * OnMessage
    收到客户端消息的回调
 * OnClose
    关闭连接,可在此处清理会话信息
    
 ### 安装
 `go get -u github.com/irealing/silly-code/ws`
 ### 示例
 
 ```go

package main

import (
	"context"
	"errors"
	"github.com/irealing/silly-code/ws"
	"log"
	"net/http"
	"strings"
)

// implements ws.SessionHook
type hook struct {
}

// call before Accept
func (h hook) BeforeAccept(r *http.Request) error {
	if strings.HasPrefix(r.RemoteAddr, "127.0.0.1") {
		// block connection from localhost
		return errors.New("deny")
	}
	return nil
}

// call OnAccept
func (h hook) OnAccept(session *ws.Session, r *http.Request) error {
	return nil
}

// OnMessage
func (h hook) OnMessage(session *ws.Session, message ws.Message) error {
	log.Printf("receive message from session %d", session.ID())
	return nil
}

// OnClose
func (h hook) OnClose(session *ws.Session) error {
	log.Print("close session", session.ID())
	// clean session storage here
	return nil
}

func main() {
	wsServer := ws.NewServer(context.Background(), &hook{}, 3, 3, 60)
	http.HandleFunc("/wsServer", func(writer http.ResponseWriter, request *http.Request) {
		if err := wsServer.Accept(writer, request); err != nil {
			log.Print(err)
		}
	})
	http.ListenAndServe(":8080", nil)
}

```
