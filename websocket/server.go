package websocket

import (
	"net/http"
	//"time"

	log "github.com/cihub/seelog"

	"github.com/gorilla/websocket"
	"github.com/xjtdy888/netio/transport"
)

type Server struct {
	callback  transport.Callback
	conn      *websocket.Conn
}

func NewServer(w http.ResponseWriter, r *http.Request, callback transport.Callback) (transport.Server, error) {
	conn, err := websocket.Upgrade(w, r, nil, 10240, 10240)
	if err != nil {
		return nil, err
	}

	ret := &Server{
		callback:  callback,
		conn:      conn,
	}

	go ret.serveHTTP(w, r)

	return ret, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
}

func (s *Server) Close() error {
	return s.conn.Close()
}

func (s *Server) writer(closeChan chan bool) {
	senderChan := s.callback.SenderChan()
	loop:
	for {
		var data []byte
		var ok bool
		select {
		case data, ok = <-senderChan:
			if ok {
				s.callback.OnRawDispatchRemote(data)
				err := s.conn.WriteMessage(websocket.TextMessage, data)
				if err != nil {
					log.Errorf("%s", err)
					break loop
				}
			}
		}
	}
	<- closeChan
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	defer s.callback.OnClose(s)
	
	closeChan := make(chan bool)
	go s.writer(closeChan)

	s.conn.SetReadLimit(1024)

	for {
		//s.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		t, p, e := s.conn.ReadMessage()

		if e != nil {
			log.Errorf("conn.ReadMessage %s", e)
			s.conn.Close()
			return
		}

		switch t {
		case websocket.TextMessage:
			s.callback.OnRawMessage(p)
		case websocket.BinaryMessage:
			log.Warnf("conn.ReadMessage type=BinaryMessage")
		}
	}
	closeChan <- true
}
