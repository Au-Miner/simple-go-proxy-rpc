package main

import (
	"fmt"
	"github.com/ankur-anand/simple-go-rpc/src/proxy"
	"net"
	"time"

	"github.com/ankur-anand/simple-go-rpc/src/server"
)

type TestClient struct {
	Ping  func() (string, error)
	Hello func() (string, error)
}

type TestServer struct{}

func (ts *TestServer) Ping() (string, error) {
	fmt.Println("server发送了pong")
	return "pong", nil
}

func (ts *TestServer) Hello() (string, error) {
	fmt.Println("server发送了World")
	return "World", nil
}

func main() {
	addr := "localhost:3212"
	srv := server.NewServer(addr)
	// srv.Register("QueryUser", QueryUser)
	srv.Register(&TestServer{})
	go srv.Run()
	time.Sleep(1 * time.Second)
	conn, _ := net.Dial("tcp", addr)

	test := &TestClient{}
	invocationProxy := proxy.NewInvocationProxy(conn)
	invocationProxy.NewProxyInstance(test)
	res, _ := test.Ping()
	fmt.Println("结果是: ", res)
	res, _ = test.Hello()
	fmt.Println("结果是: ", res)
}
