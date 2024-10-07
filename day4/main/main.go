package main

import (
	"context"
	"log"
	"net"
	"time"

	"geerpc"
)

// 定义结构体 Foo 和方法 Sum
type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func startServer(addr chan string) {
	var foo Foo
	if err := geerpc.Register(&foo); err != nil {
		log.Fatal("register error:", err)
	}
	// 随机选择端口监听
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	geerpc.Accept(l)
}

func main() {
	log.SetFlags(0) //不输出时间
	addr := make(chan string)
	go startServer(addr)

	client, _ := geerpc.Dial("tcp", <-addr)
	defer func() {
		_ = client.Close()
	}()

	time.Sleep(time.Second)
	// 服务端向客户端发送请求
	//var wg sync.WaitGroup
	//for i := 0; i < 5; i++ {
	//	wg.Add(1)
	//	go func(i int) {
	//		defer wg.Done()
	//		//args := fmt.Sprintf("geerpc req %d", i)
	//		//var reply string
	//		args := &Args{Num1: i, Num2: i * i}
	//		var reply int
	//		if err := client.Call("Foo.Sum", args, &reply); err != nil {
	//			log.Fatal("call Foo.Sum error:", err)
	//		}
	//		//log.Println("reply:", reply)
	//		log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
	//	}(i)
	//}
	//wg.Wait()

	//args := fmt.Sprintf("geerpc req %d", 11)
	args := &Args{Num1: 1, Num2: 3}
	var reply int
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	if err := client.Call(ctx, "Foo.Sum", args, &reply); err != nil {
		log.Fatal("call Foo.Sum error:", err)
	}
	log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)

}
