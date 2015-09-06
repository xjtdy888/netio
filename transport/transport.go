package transport

import (
	"net/http"

)

type Callback interface {
	SenderChan() chan []byte
	
	OnRawMessage(data []byte)
	OnRawDispatchRemote(data []byte)
	OnClose(server Server)
}

type Creater struct {
	Name      string
	Upgrading bool
	Server    func(w http.ResponseWriter, r *http.Request, callback Callback) (Server, error)
}

// Server is a transport layer in server to connect client.
type Server interface {

	// ServeHTTP handles the http request. It will call conn.onPacket when receive packet.
	ServeHTTP(http.ResponseWriter, *http.Request)

	// Close closes the transport.
	Close() error
	
}


