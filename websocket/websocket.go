package websocket

import (
	"qudao.com/tech/netio/transport"
)

var Creater = transport.Creater{
	Name:      "websocket",
	Upgrading: true,
	Server:    NewServer,
}
