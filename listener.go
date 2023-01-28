package proxy_server

import (
	"net"

	"github.com/lihongbin99/utils"

	"github.com/lihongbin99/log"
)

type Listener struct {
	net.Listener
	proxyCh chan *Proxy
	isRun   bool
	err     error
}

func ListenerNew(listen net.Listener) *Listener {
	l := &Listener{
		Listener: listen,
		proxyCh:  make(chan *Proxy, 8),
		isRun:    false,
		err:      nil,
	}

	go func(l *Listener) {
		for {
			conn, err := l.Listener.Accept()
			if err != nil {
				l.err = err
				l.proxyCh <- nil
				break
			}

			go func(l *Listener, c net.Conn) {
				// 偷窥代理协议
				conn := utils.PeepIo{Conn: c}
				protocol, err := conn.PeepN(1)
				if err != nil {
					c.Close()
					return
				}

				var proxy *Proxy = nil
				// 判断代理协议
				switch protocol[0] {
				case 4:
					log.Trace("代理 SOCKS4 协议")
					proxy = Socks4ServerProxy(&conn)
				case 5:
					log.Trace("代理 SOCKS5 协议")
					proxy = Socks5ServerProxy(&conn)
				case 22:
					log.Trace("代理 HTTPS 协议")
					proxy = HttpsServerProxy(&conn)
				default:
					log.Trace("代理 HTTP 协议: ", string(protocol[0]))
					proxy = HttpServerProxy(&conn)
				}

				l.proxyCh <- proxy
			}(l, conn)
		}
		l.Close()
	}(l)

	return l
}

func (t *Listener) Accept() (*Proxy, error) {
	return <-t.proxyCh, t.err
}
