package impl

import (
	"errors"
	"github.com/gorilla/websocket"
	"sync"
)

//封装一个websocket长链接
type Connection struct {
	wsConn    *websocket.Conn
	inChan    chan []byte
	outChan   chan []byte
	closeChan chan byte

	mutex   sync.Mutex
	isclose bool
}

//封装长链接
func InitConnection(wsConn *websocket.Conn) (conn *Connection, err error) {
	conn = &Connection{
		wsConn:    wsConn,
		inChan:    make(chan []byte, 1000),
		outChan:   make(chan []byte, 1000),
		closeChan: make(chan byte, 1),
	}
	//启动一个读go程
	go conn.readLoop()
	//启动一个写go程
	go conn.WriteLoop()

	return
}

//封装API
//从in channel中读数据
func (conn *Connection) ReadMessage() (data []byte, err error) {
	//需要检测conn是否关闭
	select {
	case data = <-conn.inChan:
	case <-conn.closeChan:
		err = errors.New("connection is close")
	}
	return
}

//将数据从out channel中读出来
func (conn *Connection) WriteMessage(data []byte) (err error) {
	//需要检测conn是否关闭
	select {
	case conn.outChan <- data:
	case <-conn.closeChan:
		err = errors.New("connection is close")
	}
	return
}

//关闭底层websocket conn
func (conn *Connection) Close() {
	//线程安全的close，可以多次被调用
	conn.wsConn.Close()

	//关闭closechan 触发case信号，使read或write go程结束，释放channel
	conn.mutex.Lock() //加锁保证go程安全,只能有一个程序操作，不会同时发生
	if !conn.isclose { //保证关闭channel只触发一次
		close(conn.closeChan)
		conn.isclose = true
	}
	conn.mutex.Unlock()
}

//内部实现
//从websocket连接中读取数据，写入in channel中
func (conn *Connection) readLoop() {
	for {
		_, data, err := conn.wsConn.ReadMessage()
		if err != nil {
			goto ERR
		}
		//防止writeloop网络异常关闭conn时，in channel还在阻塞，使用select进行监听
		select {
		case conn.inChan <- data:
		case <-conn.closeChan:
			//调用关闭conn方法,跳转结束该go程
			goto ERR
		}
	}
ERR:
	conn.Close()
}

//从out channel中读数据
func (conn *Connection) WriteLoop() {
	var data []byte
	for {
		select {
		case data = <-conn.outChan:
		case <-conn.closeChan:
			//调用关闭conn方法,跳转结束该go程
			goto ERR
		}
		if err := conn.wsConn.WriteMessage(websocket.TextMessage, data); err != nil {
			conn.Close()
		}
	}
ERR:
	conn.Close()
}
