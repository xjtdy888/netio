package store

type Message struct {
	Id string	`json:"id"`
	Data []byte	`json:"data"`
}