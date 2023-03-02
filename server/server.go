package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int
	// 在线用户的列表
	OnlineMap map[string]*User
	mapLock   sync.RWMutex

	// 消息广播的channel
	Message chan string
}

// NewServer 创建一个server接口
func NewServer(ip string, port int) *Server {
	s := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return s
}

// 启动服务
func (s *Server) Start() {
	// socket listen
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Ip, s.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	// socket close
	defer listen.Close()

	//启动监听Message的goroutine
	go s.ListenMessage()

	for true {
		// accept
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("listen.accept err:", err)
			continue
		}
		// do handler
		go s.Handler(conn)
	}

}

// 广播消息方法
func (s *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	s.Message <- sendMsg
}

func (s *Server) Handler(conn net.Conn) {
	//fmt.Println("连接成功")
	user := NewUser(conn, s)
	// 用户上线
	user.Online()
	isLive := make(chan bool)
	// 接收客户端传递发送的消息,并进行广播
	go func() {
		buf := make([]byte, 4096)
		for true {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("Conn Read err:", err)
				return
			}
			//提取用户发送的消息（去除'\n'）
			msg := string(buf[:n-1])
			//将得到的消息进行广播
			user.DoMessage(msg)
			//该用户活跃
			isLive <- true
		}
	}()

	for true {
		select {
		case <-isLive:
		//当前用户是活跃的，应该重置定时器
		//不做任何事情，为了激活select，更新下面的定时器
		case <-time.After(time.Second * 600):
			//已经超时
			//将当前的User强制关闭
			user.SendMsg("you are overtime, Forced offline\n")
			//销毁用的资源
			close(user.C)
			//关闭连接
			user.conn.Close()
			// 退出当前的handler
			return
		}
	}
	// 当前handler阻塞
	select {}
}

// 监听Message广播消息的goroutine,一旦有消息就全部发送给在线User
func (s *Server) ListenMessage() {
	for true {
		msg := <-s.Message
		//将Message全部发送给在线User
		s.mapLock.Lock()
		for _, cli := range s.OnlineMap {
			cli.C <- msg
		}
		s.mapLock.Unlock()
	}
}
