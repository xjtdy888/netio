package netio

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
	//	"github.com/kr/pretty"
	log "github.com/cihub/seelog"
	"qudao.com/tech/netio/transport"
)

// Conn is the connection object of engine.io.
type Conn interface {

	// Id returns the session id of connection.
	Id() string

	// Request returns the first http request when established connection.
	Request() *http.Request

	// Close closes the connection.
	Close() error

	Of(name string) *NameSpace

	io.Writer
}

var NotConnected = errors.New("not connected")
var ClosedError = errors.New("closed")

type transportCreaters map[string]transport.Creater

func (c transportCreaters) Get(name string) transport.Creater {
	return c[name]
}

func (c transportCreaters) Names() []string {
	names := make([]string, 0, len(c))
	for k, _ := range c {
		names = append(names, k)
	}
	return names
}

type serverCallback interface {
	configure() config
	transports() transportCreaters
	onClose(sid string)
	getEmitter(name string) *EventEmitter

	Stats() *StatsCollector
}

type state int

const (
	stateUnknow state = iota
	stateNormal
	stateUpgrading
	stateClosing
	stateClosed
)

type serverConn struct {
	id              string
	request         *http.Request
	callback        serverCallback
	transportLocker sync.RWMutex
	currentName     string
	current         transport.Server
	upgradingName   string
	upgrading       transport.Server
	state           state
	stateLocker     sync.RWMutex
	in              chan []byte
	senderChan      chan []byte

	pingInterval     time.Duration
	pingTimeout		 time.Duration
	ping chan bool
	missedHeartbeats int32
	

	nameSpaces map[string]*NameSpace
	defaultNS  *NameSpace
}

var InvalidError = errors.New("invalid transport")

func newServerConn(id string, w http.ResponseWriter, r *http.Request, callback serverCallback) (*serverConn, error) {
	/*transportName := req.Transport
	creater := callback.transports().Get(transportName)
	if creater.Name == "" {
		return nil, InvalidError
	}*/

	ret := &serverConn{
		id:           id,
		request:      r,
		callback:     callback,
		state:        stateNormal,
		in:           make(chan []byte),
		senderChan:   make(chan []byte, 0),
		pingInterval: callback.configure().PingInterval,
		pingTimeout: callback.configure().PingTimeout,
		nameSpaces:   make(map[string]*NameSpace),
	}

	/*transport, err := creater.Server(w, r, ret)
	if err != nil {
		return nil, err
	}
	ret.setCurrent(transportName, transport) */

	ret.ping = ret.pingLoop()
	go ret.infinityQueue(ret.in, ret.senderChan)
	ret.defaultNS = ret.Of("")
	ret.onOpen()

	return ret, nil
}

func (c *serverConn) Id() string {
	return c.id
}

func (c *serverConn) Request() *http.Request {
	return c.request
}

func (c *serverConn) Close() error {
	if c.getState() != stateNormal && c.getState() != stateUpgrading {
		return nil
	}
	if c.upgrading != nil {
		c.upgrading.Close()
	}
	
	for _, ns := range c.nameSpaces {
		ns.onDisconnect()
	}
	c.defaultNS.emit("close", c.defaultNS, nil)
	

	close(c.ping)
	close(c.in)				//关闭In会让InifityQueue队列退出
	c.setState(stateClosing)
	return nil
}

func (c *serverConn) OnClose(server transport.Server) {
	log.Debugf("[%s] OnClose", c.Id())
	if server != nil {
		if t := c.getUpgrade(); server == t {
			c.setUpgrading("", nil)
			t.Close()
			return
		}
		
		t := c.getCurrent()
		if server != t {
			return
		}
		t.Close()
		
		if t := c.getUpgrade(); t != nil {
			t.Close()
			c.setUpgrading("", nil)
		}
	}
	
	c.setState(stateClosed)
	c.callback.onClose(c.id)
}


func (c *serverConn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := checkRequest(r)
	transportName := req.Transport
	
	if c.getCurrent() == nil {
		creater := c.callback.transports().Get(transportName)
		transport, err := creater.Server(w, r, c)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid transport %s", transportName), http.StatusBadRequest)
			return
		}
		c.setCurrent(transportName, transport)
	}
	
	if c.currentName != transportName {
		creater := c.callback.transports().Get(transportName)
		if creater.Name == "" {
			http.Error(w, fmt.Sprintf("invalid transport %s", transportName), http.StatusBadRequest)
			return
		}
		u, err := creater.Server(w, r, c)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		c.setUpgrading(creater.Name, u)
		c.upgraded()
	}

	c.current.ServeHTTP(w, r)
}
func (c *serverConn) OnRawDispatchRemote(data []byte) {
	log.Tracef("[%s]<<< %s", c.Id(), string(data))

	sl := len(data)
	c.callback.Stats().PacketsSentPs.add(int64(sl))
}
func (c *serverConn) OnRawMessage(data []byte) {
	log.Tracef("[%s]>>> %s", c.Id(), string(data))

	sl := len(data)
	c.callback.Stats().PacketsRecvPs.add(int64(sl))

	packets := decodeFrames(data)
	for _, msg := range packets {
		c.OnRawPacket(msg)
	}

}
func (c *serverConn) OnRawPacket(data []byte) error {
	packet, err := decodePacket(data)
	if err != nil {
		log.Error("decodePacket error ", err, string(data))
		return nil
	}
	if packet == nil {
		log.Info("packet == nil ")
		return nil
	}

	if packet.EndPoint() == "" {
		if err := c.OnPacket(packet); err != nil {
			log.Error(err)
			return nil
		}
	}

	ns := c.Of(packet.EndPoint())
	if ns == nil {
		return nil
	}
	ns.onPacket(packet)
	return nil
}

func (c *serverConn) OnPacket(packet Packet) error {
	if s := c.getState(); s != stateNormal && s != stateUpgrading {
		return nil
	}
	switch packet.(type) {
	case *heartbeatPacket:
		atomic.StoreInt32(&c.missedHeartbeats, 0)
	case *disconnectPacket:
		{
			if packet.EndPoint() == "" {
				c.Close()
				return nil
			}
			ns := c.Of(packet.EndPoint())
			if ns == nil {
				return nil
			}
			ns.onDisconnect()
		}
	}

	return nil
	//c.Publish(c.namespace("packet"), data)
}

func (s *serverConn) onOpen() error {

	packet := new(connectPacket)
	s.defaultNS.connected = true
	err := s.defaultNS.sendPacket(packet)
	s.defaultNS.emit("connect", s.defaultNS, nil)

	return err
}

/*func (s *serverConn) namespace(name string) string {
		return fmt.Sprintf("conn.%s-%s", name, s.Id())
}
func (s *serverConn) Publish(name string, data []byte) error {
	return s.callback.Publish(s.namespace(name), data)
}
func (s *serverConn) Subscribe(subj string, cb store.MsgHandler) (store.Subscription, error){
	return s.callback.Subscribe(s.namespace(subj), cb)
}*/

func (s *serverConn) SenderChan() chan []byte {
	return s.senderChan
}

func (c *serverConn) getCurrent() transport.Server {
	c.transportLocker.RLock()
	defer c.transportLocker.RUnlock()

	return c.current
}

func (c *serverConn) getUpgrade() transport.Server {
	c.transportLocker.RLock()
	defer c.transportLocker.RUnlock()

	return c.upgrading
}

func (c *serverConn) setCurrent(name string, s transport.Server) {
	c.transportLocker.Lock()
	defer c.transportLocker.Unlock()

	c.currentName = name
	c.current = s
}

func (c *serverConn) setUpgrading(name string, s transport.Server) {
	c.transportLocker.Lock()
	defer c.transportLocker.Unlock()

	c.upgradingName = name
	c.upgrading = s
	c.setState(stateUpgrading)
}

func (c *serverConn) upgraded() {
	c.transportLocker.Lock()

	current := c.current
	c.current = c.upgrading
	c.currentName = c.upgradingName
	c.upgrading = nil
	c.upgradingName = ""

	c.transportLocker.Unlock()

	current.Close()
	c.setState(stateNormal)
}

func (c *serverConn) getState() state {
	c.stateLocker.RLock()
	defer c.stateLocker.RUnlock()
	return c.state
}

func (c *serverConn) setState(state state) {
	c.stateLocker.Lock()
	defer c.stateLocker.Unlock()
	c.state = state
}

func (c *serverConn) Of(name string) (nameSpace *NameSpace) {
	if nameSpace = c.nameSpaces[name]; nameSpace == nil {
		ee := c.callback.getEmitter(name)
		nameSpace = NewNameSpace(c, name, ee)
		c.nameSpaces[name] = nameSpace
	}
	return
}


func (c *serverConn) Write(p []byte) (n int, err error) {
	for {
		if c.getState() == stateClosed {
			return 0, ClosedError
		}
		select {
		case c.in <- p :
			return len(p), nil
		case <- time.After(1 * time.Second) : {}
		}
	}
	return 0, ClosedError
}

func (c *serverConn) pingLoop() chan bool {
	ticker := time.NewTicker(c.pingInterval)
	ping := make(chan bool)
	go func(){
		for {
			select {
			case <-ticker.C:
				{
					c.defaultNS.sendPacket(new(heartbeatPacket))
					n := atomic.AddInt32(&c.missedHeartbeats, 1)
	
					// TODO: Configurable
					if n > 2 {
						log.Info("heartBeat missedHeartbeats ", c.Id())
						ticker.Stop()
						c.Close()
						return
					}
				}

			case <-ping :
				ticker.Stop()
				return 
			}
		}
	}()
	return ping
	
}

func (c *serverConn) infinityQueue(in <-chan []byte, next chan<- []byte) {
	defer close(next)

	// pending events (this is the "infinite" part)
	pending := make([][]byte, 0)

recv:
	for {
		// Ensure that pending always has values so the select can
		// multiplex between the receiver and sender properly
		if len(pending) == 0 {
			v, ok := <-in
			if !ok {
				// in is closed, flush values
				break
			}
			// We now have something to send
			pending = append(pending, v)
		}

		select {
		// Queue incoming values
		case v, ok := <-in:
			if !ok {
				// in is closed, flush values
				log.Debugf("[%s] chan[in] closed", c.Id())
				break recv
			}
			pending = append(pending, v)

		// Send queued values
		case next <- encodePayload(pending):
			pending = pending[:0]
		}
	}
	if c.getCurrent() == nil {
		log.Debugf("[%s] uninitialized transport, immediately onClose", c.Id())
		c.OnClose(nil)
		return 
	}
	if len(pending) > 0 {
		select {
		case next <- encodePayload(pending):
			pending = pending[:0]
			log.Debugf("[%s] Sending the last data and close transport", c.Id())
		case <-time.After(c.pingTimeout):
		}
	}
	transport := c.getCurrent()
	if err := transport.Close(); err != nil {
		c.OnClose(transport)
	}
}
