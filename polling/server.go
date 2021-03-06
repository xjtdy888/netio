package polling

import (
	"time"
	"encoding/json"
	"strings"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"sync"

	log "github.com/cihub/seelog"
	"github.com/xjtdy888/netio/transport"
)


type state int

const (
	stateUnknow state = iota
	stateNormal
	stateClosing
	stateClosed
)

type Polling struct {
	callback    transport.Callback
	getLocker   *Locker
	postLocker  *Locker
	state       state
	stateLocker sync.Mutex
	closeChan   chan bool
	curreq		*http.Request
}

func NewServer(w http.ResponseWriter, r *http.Request, callback transport.Callback) (transport.Server, error) {
	ret := &Polling{
		callback:   callback,
		getLocker:  NewLocker(),
		postLocker: NewLocker(),
		state:      stateNormal,
		closeChan:  make(chan bool),
	}
	return ret, nil
}

func (p *Polling) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("origin") != "" {
		// https://developer.mozilla.org/En/HTTP_Access_Control
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	switch r.Method {
	case "GET":
		p.get(w, r)
	case "POST":
		p.post(w, r)
	}
}

func (p *Polling) Close() error {
	if p.getState() != stateNormal {
		return nil
	}
	close(p.closeChan)
	p.setState(stateClosing)
	if p.getLocker.TryLock() {
		if p.postLocker.TryLock() {
			p.callback.OnClose(p)
			p.setState(stateClosed)
			p.postLocker.Unlock()
		}
		p.getLocker.Unlock()
	}
	return nil
}

func (p *Polling) get(w http.ResponseWriter, r *http.Request) {
	if !p.getLocker.TryLock() {
		//http.Error(w, "8:::::overlay get", http.StatusBadRequest)
		<- time.After(1 * time.Second)
		addr, uri := "-", "-"
		if p.curreq != nil {
			addr, uri = p.curreq.RemoteAddr, p.curreq.URL.RequestURI()
		}
		fmt.Fprintf(w, "8:::::overlay get [%s] (%s)", addr, uri)
		return
	}
	p.curreq = r
	if p.getState() != stateNormal {
		http.Error(w, "closed", http.StatusForbidden)
		return
	}

	defer func() {
		if p.getState() == stateClosing {
			if p.postLocker.TryLock() {
				p.setState(stateClosed)
				p.callback.OnClose(p)
				p.postLocker.Unlock()
			}
		}
		p.getLocker.Unlock()
	}()

	closeNotifier, _ := w.(http.CloseNotifier)
	senderChan := p.callback.SenderChan()
	var data []byte
	
	select {
	case node, ok := <-senderChan:
		{
			if ok {
				data = node
			}
		}
	case <-closeNotifier.CloseNotify():
		log.Debugf("[%s] CloseNotifier ", r.URL.Path)
		return 
	case <-p.closeChan:

		return
	}

	p.callback.OnRawDispatchRemote(data)

	if strings.Contains(r.RequestURI, "/jsonp-polling/") {
		index := r.URL.Query().Get("i")
		jd, err := json.Marshal(string(data))
		if err != nil {
			log.Errorf("[%s] json.Marshal error [%s]", r.URL.Path, err)
			return 
		}
		message := fmt.Sprintf("io.j[%s](%s)", index, string(jd))
		w.Header().Set("Content-Type", "text/javascript; charset=UTF-8")
        w.Header().Set("Content-Length", fmt.Sprintf("%d", len(message)))
        w.Header().Set("X-XSS-Protection", "0")
        w.Header().Set("Connection", "Keep-Alive")
		w.Write([]byte(message))
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		
		w.Write(data)
	}
}

func (p *Polling) post(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	if !p.postLocker.TryLock() {
		http.Error(w, "8:::::::overlay post", http.StatusForbidden)
		return
	}
	if p.getState() != stateNormal {
		http.Error(w, "8::::::closed", http.StatusForbidden)
		return
	}

	defer func() {
		if p.getState() == stateClosing {
			if p.getLocker.TryLock() {
				p.setState(stateClosed)
				p.callback.OnClose(p)
				p.getLocker.Unlock()
			}
		}
		p.postLocker.Unlock()
	}()
	
	var data []byte

	if strings.Contains(r.RequestURI, "/jsonp-polling/") {
		fdata := r.PostFormValue("d")
		data = []byte(fdata)
		
		if len(data) > 0 && data[0] == '"' {
			var dedata string
			err := json.Unmarshal(data, &dedata)
			if err != nil {
				log.Errorf("[%s] json.Unmarshal error [%s]", r.URL.Path, err)
				return 
			}
			data = []byte(dedata)
		}
	}else {
		rdata, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Errorf("read post body error %s %s", err, r.URL.Path)
			return
		}
		data = rdata
		//IE XDomainRequest support
		if bytes.HasPrefix(data, []byte("data=")) {
			log.Debugf("IE XDomainRequest remove [data=]", string(data), r.URL.Path)
			data = data[5:len(data)]
		}
	}

	p.callback.OnRawMessage(data)

}

func (p *Polling) setState(s state) {
	p.stateLocker.Lock()
	defer p.stateLocker.Unlock()
	p.state = s
}

func (p *Polling) getState() state {
	p.stateLocker.Lock()
	defer p.stateLocker.Unlock()
	return p.state
}
