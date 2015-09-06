package store

import (
	"testing"
	"fmt"
	//"github.com/nats-io/nats"
)

var store *NatsPubSubStore
func init() {
	store = NewNatsPubSubStore()
}

func Test_Publish(t *testing.T) {
	type person struct {
	    Name    string
	    Address string
	    Age     int
	}
	ch := make( chan int, 0)
	
	store.Subscribe("hello", func(p *person) {
	    fmt.Printf("Received a person! %+v\n", p)
		ch <- 1
	})
	
	store.Subscribe("hello", func(subj, reply string, p *person) {
	    fmt.Printf("Received a person on subject %s! %+v\n", subj, p)
		ch <- 1
	})
	
	me := &person{Name: "derek", Age: 22, Address: "85 Second St"}
	store.Publish("hello", me)

	
	<-ch



}
