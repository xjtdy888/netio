## go 实现的socket.io 兼容0.9.x版本
该版本修改自github.com/googollee/go-engine

##目前实现传输协议
* xhr-polling
* jsonp-polling

## 计划实现
* WebSocket 传输支持

## 使用
```go
package main

import (
	"github.com/xjtdy888/netio"
	"log"
	"net/http"
	_ "net/http/pprof"

)


func main() {
	log.SetFlags(log.LstdFlags|log.Lshortfile)
	//log.SetOutput(ioutil.Discard)
	server,err := netio.NewServer(nil)
	if err != nil {
		
	}
	server.SetResourceName("socket.io")

	// Set the on connect handler
	server.On("connect", func(ns *netio.NameSpace) {
		log.Println("Connected: ", ns.Id())
		server.Broadcast("connected", ns.Id())
	})

	// Serve our website

	http.Handle("/socket.io/1/",  server)
	http.Handle("/", http.FileServer(http.Dir("./www")))
	
	log.Println("Serving at localhost:4000...")
	log.Fatal(http.ListenAndServe(":4000", nil))
}

```