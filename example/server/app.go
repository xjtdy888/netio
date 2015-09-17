package main

import (
	"qudao.com/tech/netio"
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
		server.Broadcast("connected", ns.Id())
	})

	// Set the on disconnect handler
	/*sio.On("disconnect", func(ns *socketio.NameSpace) {
		log.Println("Disconnected: ", ns.Id())
		sio.Broadcast("disconnected", ns.Id())
	})

	// Set a handler for news messages
	sio.On("news", func(ns *socketio.NameSpace, message string) {
		sio.Broadcast("news", message)
	})

	// Set a handler for ping messages
	sio.On("ping", func(ns *socketio.NameSpace) {
		ns.Emit("pong", nil)
	})

	// Set an on connect handler for the pol channel
	sio.Of("/pol").On("connect", func(ns *socketio.NameSpace) {
		log.Println("Pol Connected: ", ns.Id())
	})

	// We can broadcast messages. Set a handler for news messages from the pol
	// channel
	sio.Of("/pol").On("news", func(ns *socketio.NameSpace, message string) {
		sio.In("/pol").Broadcast("news", message)
	})

	// And respond to messages! Set a handler with a response for poll messages
	// on the pol channel
	sio.Of("/pol").On("poll", func(ns *socketio.NameSpace, message string) bool {
		if strings.Contains(message, "Nixon") {
			return true
		}

		return false
	})

	// Set an on disconnect handler for the pol channel
	sio.Of("/pol").On("disconnect", func(ns *socketio.NameSpace) {
		log.Println("Pol Disconnected: ", ns.Id())
	})*/

	// Serve our website

	http.Handle("/socket.io/1/",  server)
	http.Handle("/", http.FileServer(http.Dir("./www")))
	
	log.Println("Serving at localhost:4000...")
	log.Fatal(http.ListenAndServe(":4000", nil))
}
