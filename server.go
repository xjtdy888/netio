package netio

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"qudao.com/tech/netio/syncmap"
	//"github.com/kr/pretty"
	"qudao.com/tech/netio/polling"
log "github.com/cihub/seelog"
	//"qudao.com/tech/netio/websocket"
)

type config struct {
	PingTimeout    time.Duration
	PingInterval   time.Duration
	PollingTimeout time.Duration
	MaxConnection  int
	AllowRequest   func(*http.Request) error
	AllowUpgrades  bool
	Cookie         string
	NewId          func(r *http.Request) string
	ResourceName      string
}

type IORequest struct {
	Query     url.Values
	Headers   http.Header
	Request   *http.Request
	Path      string
	Namespace string
	Protocol  int
	Transport string
	Sid       string
}

type Handshake struct {
	Namespace string		`json:"namespace"`
	Protocol  int			`json:"protocol"`
	Transport string		`json:"transport"`
	Sid       string		`json:"sid"`
	Headers   http.Header	`json:"headers"`
	Address   string		`json:"address"`
	Time      int64			`json:"time"`
	Query     string		`json:"query"`
	Url       string		`json:"url"`
	Xdomain   bool			`json:"xdomain"`
	Secure    string		`json:"secure"`
	issued    bool	
}

// Server is the server of engine.io.
type Server struct {
	config            config
	//socketChan        chan Conn
	serverSessions    Sessions
	creaters          transportCreaters
	transportNames		[]string
	currentConnection int32
	stats             *StatsCollector
	handshaken        *syncmap.SyncMap
	eventEmitters    map[string]*EventEmitter
}

// NewServer returns the server suppported given transports. If transports is nil, server will use ["polling", "websocket"] as default.
func NewServer(transports []string) (*Server, error) {
	if transports == nil {
		transports = []string{"xhr-polling","jsonp-polling"}
	}
	creaters := make(transportCreaters)
	for _, t := range transports {
		switch t {
		case "xhr-polling":
			creaters[t] = polling.XHRCreater
		case "jsonp-polling":
			creaters[t] = polling.JSONPCreater
		/*case "websocket":
			creaters[t] = websocket.Creater*/
		default:
			return nil, InvalidError
		}
	}
	
	srv := &Server{
		config: config{
			PingTimeout:    60000 * time.Millisecond,
			PingInterval:   12000 * time.Millisecond,
			PollingTimeout: 20000 * time.Millisecond,
			MaxConnection:  20000,
			AllowRequest:   func(*http.Request) error { return nil },
			AllowUpgrades:  true,
			Cookie:         "io",
			NewId:          newId,
			ResourceName:      "net.io",
		},
		//socketChan:     make(chan Conn),
		serverSessions: newServerSessions(),
		creaters:       creaters,
		transportNames: 	transports,
		stats:          NewStatsCollector(),
		handshaken:     syncmap.New(),
		eventEmitters : make(map[string]*EventEmitter),
	}
	go srv.garbageCollection()
	return srv, nil
}
func (s *Server) garbageCollection() {
	for {
		<- time.After(10 * time.Second)
		keys := make([]string, 0)
		for item := range s.handshaken.IterItems() {
			value, ok := item.Value.(*Handshake)
			if !ok {
				keys = append(keys, item.Key)
				continue
			}
			if value.issued == false && (time.Now().Unix() - value.Time) > 30 {
				keys = append(keys, item.Key)
			}
		}
		for _, key := range keys {
			log.Trace("garbageCollection clean ", key)
			s.handshaken.Delete(key)
		}
	}
}
/*
func (s *Server) watchMessage() {

	watcher, err := s.Subscribe("dispatch-remote", func(sub, rep string, msg *store.Message) {
		sessionId := msg.Id
		data := msg.Data		
		conn := s.serverSessions.Get(sessionId)
		if conn != nil{
			conn.Write(data)
		}
	})
	if err != nil {
		fmt.Println(err)
		_ = watcher
	}
	
	s.Publish("dispatch-remote", &store.Message{"20000", []byte("hahaha")})
}
*/
// SetPingTimeout sets the timeout of ping. When time out, server will close connection. Default is 60s.
func (s *Server) SetPingTimeout(t time.Duration) {
	s.config.PingTimeout = t
}

// SetPingInterval sets the interval of ping. Default is 25s.
func (s *Server) SetPingInterval(t time.Duration) {
	s.config.PingInterval = t
}

// SetMaxConnection sets the max connetion. Default is 1000.
func (s *Server) SetMaxConnection(n int) {
	s.config.MaxConnection = n
}

// SetAllowRequest sets the middleware function when establish connection. If it return non-nil, connection won't be established. Default will allow all request.
func (s *Server) SetAllowRequest(f func(*http.Request) error) {
	s.config.AllowRequest = f
}

// SetAllowUpgrades sets whether server allows transport upgrade. Default is true.
func (s *Server) SetAllowUpgrades(allow bool) {
	s.config.AllowUpgrades = allow
}

// SetCookie sets the name of cookie which used by engine.io. Default is "io".
func (s *Server) SetCookie(prefix string) {
	s.config.Cookie = prefix
}

// SetNewId sets the callback func to generate new connection id. By default, id is generated from remote addr + current time stamp
func (s *Server) SetNewId(f func(*http.Request) string) {
	s.config.NewId = f
}

// SetSessionManager sets the sessions as server's session manager. Default sessions is single process manager. You can custom it as load balance.
func (s *Server) SetSessionManager(sessions Sessions) {
	s.serverSessions = sessions
}
func (s *Server) GetSessionManager() Sessions{
	return s.serverSessions
}

func (s *Server) SetResourceName(ns string) {
	s.config.ResourceName = ns
}

func (s *Server) namespace(node string) string {
	return fmt.Sprintf("%s.%s", s.config.ResourceName, node)
}

func (s *Server) onHandshake(id string, data *Handshake) {
	s.handshaken.Set(id, data)
}

func (s *Server) handshakeData(data *IORequest) *Handshake {

	return &Handshake{
		Namespace: data.Namespace,
		Protocol:  data.Protocol,
		Transport: data.Transport,
		Sid:       data.Sid,

		Headers: data.Headers,
		Address: data.Request.RemoteAddr,
		Time:    time.Now().Unix(),
		Url:     data.Request.URL.String(),
		Xdomain: data.Headers.Get("origin") != "",
		Secure:  "",
		issued:  false,
	}
}

func (s *Server) handleHandshake(ir *IORequest, w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("origin") != "" {
		// https://developer.mozilla.org/En/HTTP_Access_Control
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if err := s.config.AllowRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	handshakeData := s.handshakeData(ir)
	sid := s.config.NewId(r)

	s.onHandshake(sid, handshakeData)

	transports := s.transportNames

	data := fmt.Sprintf("%s:%d:%d:%s",
		sid,
		s.config.PingInterval/time.Second,
		s.config.PollingTimeout/time.Second,
		strings.Join(transports, ","))

	jsonp := r.URL.Query().Get("jsonp")
	if jsonp != "" {
		w.Header().Set("Content-Type", "application/javascript; charset=UTF-8")
		fmt.Fprintf(w, "io.j[%s](\"%s\");", jsonp, data)
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		fmt.Fprintf(w, "%s", data)
	}
}
func checkRequest(r *http.Request) *IORequest {
	ns := ""
	protocol := 0
	transport := ""
	sid := ""

	path := r.URL.Path
	spliter := strings.Split(path, "/")

	if len(spliter) > 1 {
		ns, _ = String(spliter[1])
	}

	if len(spliter) > 2 {
		protocol, _ = Int(spliter[2])
	}

	if len(spliter) > 3 {
		transport = spliter[3]
	}

	if len(spliter) > 4 {
		sid = spliter[4]
	}

	return &IORequest{
		Query:     r.Form,
		Headers:   r.Header,
		Request:   r,
		Path:      r.URL.Path,
		Namespace: ns,
		Protocol:  protocol,
		Sid:       sid,
		Transport: transport,
	}
}

// ServeHTTP handles http request.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	s.stats.ConnectionOpened()
	defer s.stats.ConnectionClosed()

	req := checkRequest(r)
	//pretty.Println("%#v", req)
	if req.Sid == "" {
		s.handleHandshake(req, w, r)
		return
	}

	conn := s.serverSessions.Get(req.Sid)
	if conn == nil {
		data, ok := s.handshaken.Get(req.Sid)

		if !ok {
			http.Error(w, "invalid sid", http.StatusBadRequest)
			return
		}

		handshake, ok := data.(*Handshake)

		if !ok {
			http.Error(w, "invalid handshake", http.StatusBadRequest)
		}
		handshake.issued = true
		s.handshaken.Delete(req.Sid)
		

		n := atomic.AddInt32(&s.currentConnection, 1)
		if int(n) > s.config.MaxConnection {
			atomic.AddInt32(&s.currentConnection, -1)
			http.Error(w, "too many connections", http.StatusServiceUnavailable)
			return
		}

		var err error
		conn, err = newServerConn(req, w, r, s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		s.serverSessions.Set(req.Sid, conn)

		//s.socketChan <- conn
	}
	http.SetCookie(w, &http.Cookie{
		Name:  s.config.Cookie,
		Value: req.Sid,
	})
	
	s.stats.SessionOpened()
	defer s.stats.SessionClosed()
	
	if r.Method == "POST" {
		s.stats.PacketsRecvPs.add(r.ContentLength)
	}
	
	conn.(*serverConn).ServeHTTP(w, r)
}

// Accept returns Conn when client connect to server.
/*func (s *Server) Accept() (Conn, error) {
	return <-s.socketChan, nil
}*/

func (s *Server) configure() config {
	return s.config
}

func (s *Server) transports() transportCreaters {
	return s.creaters
}

func (s *Server) Stats() *StatsCollector {
	return s.stats
}
/*
func (s *Server) Publish(name string, data interface{}) error {
	ns := s.namespace(name)
	return s.pubsub.Publish(ns, data)
}
func (s *Server) Subscribe(subj string, cb store.Handler) (store.Subscription, error){
	ns := s.namespace(subj)
	return s.pubsub.Subscribe(ns, cb)
}

func (s *Server) SubscribeAsync(subj string) (store.Subscription, error){
	ns := s.namespace(subj)
	return s.pubsub.SubscribeSync(subj)
}*/

func (s *Server) onClose(id string) {
	s.serverSessions.Remove(id)
	atomic.AddInt32(&s.currentConnection, -1)
}

func (srv *Server) Of(name string) *EventEmitter {
	ret, ok := srv.eventEmitters[name]
	if !ok {
		ret = NewEventEmitter()
		srv.eventEmitters[name] = ret
	}
	return ret
}

func (srv *Server) In(name string) *Broadcaster {
	namespaces := []*NameSpace{}
	for _, conn := range srv.serverSessions.IterItems() {
		ns := conn.Of(name)
		if ns != nil {
			namespaces = append(namespaces, ns)
		}
	}

	return &Broadcaster{Namespaces: namespaces}
}

func (srv *Server) Broadcast(name string, args ...interface{}) {
	srv.In("").Broadcast(name, args...)
}

func (srv *Server) Except(ns *NameSpace) *Broadcaster {
	return srv.In("").Except(ns)
}

func (srv *Server) On(name string, fn interface{}) error {
	return srv.Of("").On(name, fn)
}

func (srv *Server) RemoveListener(name string, fn interface{}) {
	srv.Of("").RemoveListener(name, fn)
}

func (srv *Server) RemoveAllListeners(name string) {
	srv.Of("").RemoveAllListeners(name)
}

func (srv *Server) getEmitter(name string) *EventEmitter {
	ee := srv.eventEmitters[name]
	if ee == nil {
		srv.eventEmitters[name] = NewEventEmitter()
		ee = srv.eventEmitters[name]
	}
	return ee
}


func newId(r *http.Request) string {
	hash := fmt.Sprintf("%d %s %s", rand.Uint32(), r.RemoteAddr, time.Now())
	md5sid := md5.Sum([]byte(hash))
	return fmt.Sprintf("%x", md5sid)
}
