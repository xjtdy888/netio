package store

import (
	"time"
)


type Msg struct {
    Subject string
    Reply   string
    Data    []byte
    Sub     Subscription
}

type MsgHandler func(msg *Msg)
type Handler interface{}


type Subscription interface {
	IsValid() bool
	Unsubscribe() error
	AutoUnsubscribe(max int) error 
	NextMsg(timeout time.Duration) (msg *Msg, err error)
    // Subject that represents this subscription. This can be different
    // than the received subject inside a Msg if this is a wildcard.
    Subject() string

    // Optional queue group name. If present, all subscriptions with the
    // same name will form a distributed queue, and each message will
    // only be processed by one member of the group.
    Queue() string
    // contains filtered or unexported fields
}



type PubSubStore interface {
	Publish(subj string, data interface{}) error
	Subscribe(subj string, cb Handler) (Subscription, error)
}
