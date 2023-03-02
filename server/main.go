package main

func main() {
	//初始化server
	server := NewServer("127.0.0.1", 8080)
	server.Start()
}
