package websocket

import (
	"github.com/xjtdy888/netio/transport"
)

var Creater = transport.Creater{
	Name:      "websocket",
	Upgrading: true,
	Server:    NewServer,
}
