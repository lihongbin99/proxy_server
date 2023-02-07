package proxy_server

import "net"

var (
	httpProxyAddr   string
	httpsProxyAddr  string
	socks4ProxyAddr string
	socks5ProxyAddr string
)

func SetProxy(proxyAddr string) {
	SetHttpProxy(proxyAddr)
	SetHttpsProxy(proxyAddr)
	SetSocks4Proxy(proxyAddr)
	SetSocks5Proxy(proxyAddr)
}

func SetSocksProxy(proxyAddr string) {
	SetSocks4Proxy(proxyAddr)
	SetSocks5Proxy(proxyAddr)
}

func SetHttpProxy(proxyAddr string) {
	httpProxyAddr = proxyAddr
}

func SetHttpsProxy(proxyAddr string) {
	httpsProxyAddr = proxyAddr
}

func SetSocks4Proxy(proxyAddr string) {
	socks4ProxyAddr = proxyAddr
}

func SetSocks5Proxy(proxyAddr string) {
	socks5ProxyAddr = proxyAddr
}

func TryConnectServer(hostname string, port uint16) (net.Conn, error) {
	if socks5ProxyAddr != "" {
		return Socks5(hostname, port)
	}
	if socks4ProxyAddr != "" {
		//
	}
	if httpsProxyAddr != "" {
		//
	}
	if httpProxyAddr != "" {
		//
	}
	return nil, nil
}
