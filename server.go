package proxy_server

import (
	"net"

	"github.com/lihongbin99/utils"

	"github.com/lihongbin99/log"
)

func Listen(address string) (*Listener, error) {
	listen, err := net.Listen("tcp", address)
	return ListenerNew(listen), err
}

func ListenAndServer(address string) error {
	listen, err := Listen(address)
	if err != nil {
		return err
	}

	log.Debug("The server started successfully: ", address)
	for {
		proxy, err := listen.Accept()
		if err != nil {
			log.Error(err)
			break
		}

		if proxy.Err != nil {
			log.Debug(proxy.Err)
			continue
		}

		go utils.Tunnel(proxy.ClientConn, proxy.ServerConn)
	}
	return nil
}
