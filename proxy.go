package proxy_server

import "net"

type Proxy struct {
	ClientConn net.Conn
	ServerConn net.Conn

	IP       []byte // len(4)
	Port     uint16
	HOSTNAME string

	Err error
}

func (p *Proxy) Error(err error) *Proxy {
	p.ClientConn.Close()
	p.Err = err
	return p
}