package polling

import (
	"qudao.com/tech/netio/transport"
)

var XHRCreater = transport.Creater{
	Name:      "xhr-polling",
	Upgrading: false,
	Server:    NewServer,
}

var JSONPCreater = transport.Creater{
	Name:      "jsonp-polling",
	Upgrading: false,
	Server:    NewServer,
}
