package main

import (
	"net"
	"strings"
)

type User struct {
	Name   string
	Addr   string
	C      chan string // 传输数据的管道
	conn   net.Conn    //当前用户的连接
	server *Server     // 当前用户属于哪个server
}

// 创建一个用户API
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name:   userAddr,
		Addr:   userAddr,
		C:      make(chan string),
		conn:   conn,
		server: server,
	}
	// 启动监听当前user channel消息的goroutine
	go user.ListenMessage()

	return user
}

// 用户上线业务
func (u *User) Online() {
	// 当前用户上线,将用户加入到onlineMap中
	u.server.mapLock.Lock()
	u.server.OnlineMap[u.Name] = u
	u.server.mapLock.Unlock()

	// 广播当前用户上线消息
	u.server.BroadCast(u, "online")
}

// 用户上线业务
func (u *User) Offline() {
	// 当前用户上线,将用户从onlineMap中删除
	u.server.mapLock.Lock()
	delete(u.server.OnlineMap, u.Name)
	u.server.mapLock.Unlock()

	// 广播当前用户上线消息
	u.server.BroadCast(u, "offline")
}

// 给当前user对应的客户端发消息
func (u User) SendMsg(msg string) {
	u.conn.Write([]byte(msg))
}

// 用户处理消息业务
func (u *User) DoMessage(msg string) {
	if msg == "who" {
		// 查询当前在线用户
		u.server.mapLock.Lock()
		for _, user := range u.server.OnlineMap {
			onlineMsg := "[" + user.Addr + "]" + user.Name + ":" + "online...\n"
			u.SendMsg(onlineMsg)
		}
		u.server.mapLock.Unlock()

	} else if len(msg) > 7 && msg[:7] == "rename|" {
		// 消息格式：rename|张三
		newName := strings.Split(msg, "|")[1]
		// 判断name是否存在
		_, ok := u.server.OnlineMap[newName]
		if ok {
			u.SendMsg("this name is already register!\n")
		} else {
			u.server.mapLock.Lock()
			delete(u.server.OnlineMap, u.Name)
			u.server.OnlineMap[newName] = u
			u.server.mapLock.Unlock()
			u.Name = newName
			u.SendMsg("your new name: " + u.Name + "\n")
		}
	} else if len(msg) > 4 && msg[:3] == "to|" {
		// 消息格式：to|张三|消息内容
		// 1.获取对方的用户名
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			u.SendMsg("the msg format is not valid, please use: to|name|content...\n")
			return
		}
		// 2.根据用户名的到对方的User对象
		remoteUser, ok := u.server.OnlineMap[remoteName]
		if !ok {
			u.SendMsg("this username is null in register\n")
			return
		}
		// 3.获取消息内容，通过对方的User对象将消息内容发送过去
		content := strings.Split(msg, "|")[2]
		if content == "" {
			u.SendMsg("this msg is null, please repeat send!\n")
			return
		}
		remoteUser.SendMsg(u.Name + "say: " + content + "\n")
	} else {
		u.server.BroadCast(u, msg)
	}
}

// 监听当前User channel的方法，一旦有消息就发送给对应客户端
func (u *User) ListenMessage() {
	for true {
		msg := <-u.C
		u.conn.Write([]byte(msg + "\n"))
	}
}
