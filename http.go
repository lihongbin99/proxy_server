package proxy_server

import (
	"fmt"
	"net"
)

func HttpServerProxy(conn net.Conn) *Proxy {
	conn.Close()
	return &Proxy{
		ClientConn: conn,
		Err:        fmt.Errorf("The http proxy protocol is not supported"),
	}
}
