package proxy_server

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/lihongbin99/utils"

	"github.com/lihongbin99/log"
)

type Socks5Req struct {
	VER      byte
	CMD      byte
	RSV      byte
	ATYP     byte
	DSTADDR  []byte
	DSTPORT  uint16
	HOSTNAME string
}

type Socks5Rep struct {
	VER     byte
	REP     byte
	RSV     byte
	ATYP    byte
	BNDADDR []byte
	BNDPORT uint16
}

func Socks5ServerProxy(conn net.Conn) *Proxy {
	var err error

	proxy := Proxy{
		ClientConn: conn,
		Err:        nil,
	}

	c := utils.BaseIO{Conn: conn}

	var VER [1]byte
	if err = c.ReadN(VER[:], len(VER)); err != nil {
		return proxy.Error(fmt.Errorf("socks5 read request protocol exception: %v", err))
	}
	if VER[0] != 5 {
		return proxy.Error(fmt.Errorf("This request is not a socks5 protocol"))
	}

	// 验证身份
	if err = Socks5Authenticate(conn); err != nil {
		return proxy.Error(err)
	}

	var buf [4]byte
	if err = c.ReadN(buf[:], len(buf)); err != nil {
		return proxy.Error(fmt.Errorf("socks5 read request exception: %v", err))
	}
	if buf[0] != 5 {
		return proxy.Error(fmt.Errorf("This request is not a socks5 protocol"))
	}

	req := Socks5Req{}
	req.VER = buf[0]
	req.CMD = buf[1]
	req.RSV = buf[2]
	req.ATYP = buf[3]

	// 读取ADDR
	var DSTADDR []byte
	switch req.ATYP {
	case 1: // IPv4地址
		DSTADDR = make([]byte, 4)
	case 3: // 域名
		var domainNameLen [1]byte
		if err = c.ReadN(domainNameLen[:], len(domainNameLen)); err != nil {
			return proxy.Error(fmt.Errorf("socks5 read request domain name length exception: %v", err))
		}
		DSTADDR = make([]byte, domainNameLen[0])
	case 4: // IPv6地址
		DSTADDR = make([]byte, 16)
	default:
		return proxy.Error(fmt.Errorf("socks5 does not support CD: %d", req.ATYP))
	}
	if err = c.ReadN(DSTADDR[:], len(DSTADDR)); err != nil {
		return proxy.Error(fmt.Errorf("socks5 read request DSTADDR exception: %v", err))
	}
	req.DSTADDR = DSTADDR
	switch req.ATYP {
	case 1:
		req.HOSTNAME = net.IP(req.DSTADDR).String()
	case 3:
		req.HOSTNAME = string(req.DSTADDR)
	case 4:
		req.HOSTNAME = net.IP(req.DSTADDR).String()
	}

	// 读取端口
	var DSTPORT [2]byte
	if err = c.ReadN(DSTPORT[:], len(DSTPORT)); err != nil {
		return proxy.Error(fmt.Errorf("socks5 read request DSTPORT exception: %v", err))
	}
	req.DSTPORT = uint16(DSTPORT[0])<<8 | uint16(DSTPORT[1])

	log.Debug("socks5 ---> VER:", req.VER,
		"; CMD:", req.CMD,
		"; RSV:", req.RSV,
		"; ATYP:", req.ATYP,
		"; DSTADDR:", req.DSTADDR,
		"; DSTPORT:", req.DSTPORT,
		"; HOSTNAME:", req.HOSTNAME)

	var rep *Socks5Rep
	switch req.CMD {
	case 1: // CONNECT
		rep, proxy.Err = Socks5Connect(&proxy, &req)
	case 2: // BIND
		rep, proxy.Err = Socks5Bind(&proxy, &req)
	// case 3: // UDP
	default:
		proxy.Err = fmt.Errorf("socks5 does not support CMD: %d", req.CMD)
	}

	repb := Socks5MakeRepBuf(rep)
	log.Trace("socks5 return data:", repb)
	c.Write(repb)
	if proxy.Err != nil {
		c.Close()
	}
	return &proxy
}

func Socks5Authenticate(conn net.Conn) error {
	var err error
	c := utils.BaseIO{Conn: conn}

	var NMETHODS [1]byte
	if err = c.ReadN(NMETHODS[:], len(NMETHODS)); err != nil {
		return fmt.Errorf("socks5 read request nmethods exception: %v", err)
	}
	if NMETHODS[0] < 1 {
		return fmt.Errorf("There are too few socks5 authentication methods")
	}

	METHODS := make([]byte, NMETHODS[0])
	if err = c.ReadN(METHODS, len(METHODS)); err != nil {
		return fmt.Errorf("socks5 read request methods exception: %v", err)
	}

	// 未实现socks5认证
	index := bytes.IndexByte(METHODS, 0)
	if index < 0 {
		return fmt.Errorf("Does not support socks5 authentication: %v", METHODS)
	}

	c.Write([]byte{5, 0})
	return err
}

func Socks5Connect(proxy *Proxy, req *Socks5Req) (*Socks5Rep, error) {
	if targetConn, err := TryConnectServer(req.HOSTNAME, req.DSTPORT); err != nil {
		return Socks5MakeRep(proxy, req, 3), err
	} else if targetConn != nil {
		proxy.ServerConn = targetConn
		return Socks5MakeRep(proxy, req, 0), nil
	}

	addrStr := ""
	if strings.Index(req.HOSTNAME, ":") < 0 {
		addrStr = fmt.Sprintf("%s:%d", req.HOSTNAME, req.DSTPORT)
	} else {
		addrStr = fmt.Sprintf("[%s]:%d", req.HOSTNAME, req.DSTPORT)
	}
	addr, err := net.ResolveTCPAddr("tcp", addrStr)
	if err != nil {
		return Socks5MakeRep(proxy, req, 4), err
	}
	targetConn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		// 暂时没有判断更多的异常
		return Socks5MakeRep(proxy, req, 3), err
	}

	proxy.ServerConn = targetConn
	return Socks5MakeRep(proxy, req, 0), nil
}

func Socks5Bind(proxy *Proxy, req *Socks5Req) (*Socks5Rep, error) {
	addrStr := ""
	if strings.Index(req.HOSTNAME, ":") < 0 {
		addrStr = fmt.Sprintf("%s:%d", req.HOSTNAME, req.DSTPORT)
	} else {
		addrStr = fmt.Sprintf("[%s]:%d", req.HOSTNAME, req.DSTPORT)
	}
	addr, err := net.ResolveTCPAddr("tcp", addrStr)
	if err != nil {
		return Socks5MakeRep(proxy, req, 4), err
	}
	listen, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return Socks5MakeRep(proxy, req, 3), err
	}

	targetConn, err := listen.AcceptTCP()
	listen.Close()
	if err != nil {
		return Socks5MakeRep(proxy, req, 3), err
	}
	proxy.ServerConn = targetConn

	return Socks5MakeRep(proxy, req, 0), nil
}

func Socks5MakeRep(proxy *Proxy, req *Socks5Req, repCode byte) *Socks5Rep {
	rep := Socks5Rep{VER: 5, REP: repCode, RSV: 0}

	if proxy.ServerConn != nil {
		remoteAddr := proxy.ServerConn.RemoteAddr().String()
		ip, portStr, _ := net.SplitHostPort(remoteAddr)

		if strings.Index(ip, ":") < 0 {
			rep.ATYP = 1
			rep.BNDADDR = net.ParseIP(ip).To4()
		} else {
			rep.ATYP = 4
			rep.BNDADDR = net.ParseIP(ip).To16()
		}

		port, _ := strconv.Atoi(portStr)
		rep.BNDPORT = uint16(port)
	} else {
		rep.ATYP = req.ATYP
		rep.BNDADDR = req.DSTADDR
		rep.BNDPORT = req.DSTPORT
	}

	proxy.IP = rep.BNDADDR
	proxy.Port = rep.BNDPORT
	proxy.HOSTNAME = req.HOSTNAME
	return &rep
}

func Socks5MakeRepBuf(rep *Socks5Rep) []byte {
	var buf = make([]byte, 4, 22)
	buf[0] = rep.VER
	buf[1] = rep.REP
	buf[2] = rep.RSV
	buf[3] = rep.ATYP
	if rep.ATYP == 3 {
		buf = append(buf, byte(len(rep.BNDADDR)))
	}
	buf = append(buf, rep.BNDADDR...)
	buf = append(buf, byte(rep.BNDPORT>>8), byte(rep.BNDPORT))
	return buf
}

func Socks5(hostname string, port uint16) (net.Conn, error) {
	addr, err := net.ResolveTCPAddr("tcp", socks5ProxyAddr)
	if err != nil {
		return nil, err
	}
	proxyConn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}

	proxyConn.Write([]byte{5, 1, 0})

	var authenticateBuf [3]byte
	readLen, err := proxyConn.Read(authenticateBuf[:])
	if err != nil {
		return nil, err
	}

	if readLen != 2 {
		return nil, fmt.Errorf("Read Socks5 Protocol Error: ReadLen(%d)", readLen)
	}
	if authenticateBuf[0] != 5 {
		return nil, fmt.Errorf("Socks5 Protocol Error: Protocol(%d)", authenticateBuf[0])
	}
	if authenticateBuf[1] != 0 {
		return nil, fmt.Errorf("Socks5 Authenticate Error: Authenticate(%d)", authenticateBuf[1])
	}

	socks5 := make([]byte, 0)
	socks5 = append(socks5, 5, 1, 0, 3, byte(len(hostname)))
	socks5 = append(socks5, []byte(hostname)...)
	socks5 = append(socks5, byte(port>>8), byte(port&0xFF))
	proxyConn.Write(socks5)

	var buf [1024]byte
	readLen, err = proxyConn.Read(buf[:])
	if err != nil {
		return nil, err
	}
	if buf[0] != 5 {
		return nil, fmt.Errorf("This request is not a socks5 protocol")
	}
	if buf[1] != 0 {
		return nil, fmt.Errorf("Connect ServerError(%d)", buf[1])
	}

	return proxyConn, nil
}
