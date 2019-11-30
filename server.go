package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
	"websocket练习/impl"
)

func wsHandle(w http.ResponseWriter, r *http.Request) {
	//定义转换器
	upgrade := websocket.Upgrader{
		//函数的作用，当发现跨域请求的时候，允许跨域
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	//创建websocket连接，完成upgrade应答
	wsConn, err := upgrade.Upgrade(w, r, nil);
	if err != nil {
		return
	}
	//初始化connention
	conn, err := impl.InitConnection(wsConn)
	if err != nil {
		fmt.Println("初始化conn失败")
		goto ERR
	}

	//起go程实现心跳机制，测试客户端是否关闭
	go func() {
		for {
			if err := conn.WriteMessage([]byte("heartbeat")); err != nil {
				return
			}
			time.Sleep(time.Second)
		}
	}()

	//实现业务处理
	for {
		data, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("读取message失败", err)
			goto ERR
		}
		if err := conn.WriteMessage(data); err != nil {
			goto ERR
		}
	}
ERR:
	conn.Close()
}

//建立HTTP服务器
func main() {
	http.HandleFunc("/ws", wsHandle)
	fmt.Println("服务器启动")
	http.ListenAndServe(":8888", nil)
}
