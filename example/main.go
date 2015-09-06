package main

import (
	//"fmt"
	"log"
	"net/http"
//	"time"

	"qudao.com/tech/netio"
)

func main() {
	server, err := netio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	//server.SetPingInterval(time.Second * 2)
	//server.SetPingTimeout(time.Second * 3)

	/*go func() {
		for {
			conn, _ := server.Accept()
			fmt.Println("Accept:", conn)
		}
	}()*/

	http.Handle("/net.io/1/",  server)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	log.Println("Serving at localhost:4000...")
	log.Fatal(http.ListenAndServe(":4000", nil))
}
