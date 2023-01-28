package proxy_server

import (
	"fmt"
	"net"
)

func HttpsServerProxy(conn net.Conn) *Proxy {
	conn.Close()
	return &Proxy{
		ClientConn: conn,
		Err:        fmt.Errorf("The https proxy protocol is not supported"),
	}
}
