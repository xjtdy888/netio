package websocket

import (
	"sync"
	"net/http"

	log "github.com/cihub/seelog"

	"github.com/gorilla/websocket"
	"github.com/xjtdy888/netio/transport"
)

type state int

const (
	stateUnknow state = iota
	stateNormal
	stateClosing
	stateClosed
)

type Server struct {
	callback  transport.Callback
	conn      *websocket.Conn
	state       state
	stateLocker sync.Mutex
	broadOnce sync.Once
	closeChan   chan bool
}

func NewServer(w http.ResponseWriter, r *http.Request, callback transport.Callback) (transport.Server, error) {
	conn, err := websocket.Upgrade(w, r, nil, 10240, 10240)
	if err != nil {
		return nil, err
	}

	ret := &Server{
		callback:  callback,
		conn:      conn,
		state:      stateNormal,
		closeChan:  make(chan bool, 1),
	}

	go ret.serveHTTP(w, r)

	return ret, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
}

func (s *Server) Close() error {
	if s.getState() != stateNormal {
		return nil
	}
	s.setState(stateClosing)
	s.broadClose()
	return nil
}

func (s *Server) broadClose() error {
	s.broadOnce.Do(func(){
		close(s.closeChan)
	})
	return nil
}


func (s *Server) setState(st state) {
	s.stateLocker.Lock()
	defer s.stateLocker.Unlock()
	s.state = st
}

func (s *Server) getState() state {
	s.stateLocker.Lock()
	defer s.stateLocker.Unlock()
	return s.state
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
					s.Close()
					break loop
				}
			}
		case <-closeChan :
			break loop 
		}
	}
	log.Infof("[%s] websocket writer exiting", s.conn.RemoteAddr().String())
	s.conn.Close()
	
}

func (s *Server) reader(closeChan chan bool) {
	
	s.conn.SetReadLimit(1024)
	loop:
	for {
		//s.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		t, p, e := s.conn.ReadMessage()

		if e != nil {
			//if s.getState() == stateNormal {
				log.Errorf("conn.ReadMessage %s", e)
				s.Close()
			//}
			break loop
		}

		switch t {
		case websocket.TextMessage:
			s.callback.OnRawMessage(p)
		case websocket.BinaryMessage:
			log.Warnf("conn.ReadMessage type=BinaryMessage")
		}
		
		select {
			case <- closeChan: {
				break loop 
			}
			default: {}
		}
	}
	log.Infof("[%s] websocket reader exiting", s.conn.RemoteAddr().String())
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	defer s.callback.OnClose(s)
	
	ch := make(chan bool)
	go s.writer(ch)
	go s.reader(ch)
	
	select {
		case <- s.closeChan: {
			log.Infof("[%s] Closing", s.conn.RemoteAddr().String())
			close(ch)
		}
	}
	s.setState(stateClosed)
}
