package store

import (
	"log"
	"github.com/nats-io/nats"
	"time"
//	"github.com/kr/pretty"
	)

type NatsPubSubStore struct {
	nc *nats.Conn
	jc *nats.EncodedConn
}

func NewNatsPubSubStore() *NatsPubSubStore{
	nc, err := nats.Connect("nats://192.168.99.18:4222")
	if err != nil {
		//TODO nats connect fails handler
		log.Fatalln("nats.Connect fails", err)
	}
	jc, err := nats.NewEncodedConn(nc, "json")
	if err != nil {
		log.Fatalln("nats.NewEncodedConn fails", err)
	}
	ret := &NatsPubSubStore{
		nc : nc,
		jc : jc,
	}
	return ret
}


func (p *NatsPubSubStore) Publish(subj string, data interface{}) error {
	return p.jc.Publish(subj, data)
}
func (p *NatsPubSubStore) Subscribe(subj string, cb Handler) (Subscription, error) {
	sub, err := p.jc.Subscribe(subj, cb)
	/*sub, err := p.jc.Subscribe(subj, func(msg *nats.Msg){
		m := natsMsgWrapper(msg)
		cb(m)
	})*/
	if err != nil {
		return nil, err
	}
	wrapsub := natsSubWrapper(sub)
	return wrapsub, err
}

/*func (p *NatsPubSubStore) SubscribeSync(subj string) (Subscription, error) {
	sub, err := p.nc.SubscribeSync(subj)
	if err != nil {
		return nil, err
	}
	wrapsub := natsSubWrapper(sub)
	return wrapsub, err
}*/

func natsMsgWrapper(msg *nats.Msg) *Msg {
	m := &Msg{
			Subject : msg.Subject,
		    Reply   : msg.Reply,
		    Data    : msg.Data,
		}
	if msg.Sub != nil {
		m.Sub = natsSubWrapper(msg.Sub)
	}
	return m
}

func natsSubWrapper(sub *nats.Subscription) Subscription {
	return &NatsSubscription{sub : sub}
}


type NatsSubscription struct {
	sub *nats.Subscription
}

func (wrap *NatsSubscription) IsValid() bool {
	return wrap.sub.IsValid()
}

func (wrap *NatsSubscription) Unsubscribe() error {
	return wrap.sub.Unsubscribe()
}

func (wrap *NatsSubscription) AutoUnsubscribe(max int) error  {
	return wrap.sub.AutoUnsubscribe(max)  
}

func (wrap *NatsSubscription) NextMsg(timeout time.Duration) (msg *Msg, err error)  {
	res, err := wrap.sub.NextMsg(timeout)
	if err != nil {
		return nil, err
	}
	msg = natsMsgWrapper(res)
	return msg, err
	
}

func (wrap *NatsSubscription) Subject() string {
	return wrap.sub.Subject
}

func (wrap *NatsSubscription) Queue() string {
	return wrap.sub.Queue
}