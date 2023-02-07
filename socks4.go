package proxy_server

import (
	"fmt"
	"net"
	"strconv"

	"github.com/lihongbin99/utils"

	"github.com/lihongbin99/log"
)

type Socks4Req struct {
	VN       byte
	CD       byte
	DSTPORT  uint16
	DSTIP    []byte // len(4)
	USERID   string
	HOSTNAME string
}

type Socks4Rep struct {
	VN      byte
	CD      byte
	DSTPORT uint16
	DSTIP   []byte // len(4)
}

func Socks4ServerProxy(conn net.Conn) *Proxy {
	var err error

	proxy := Proxy{
		ClientConn: conn,
		Err:        nil,
	}

	c := utils.BaseIO{Conn: conn}

	var buf [8]byte
	if err = c.ReadN(buf[:], len(buf)); err != nil {
		return proxy.Error(fmt.Errorf("socks4 read request exception: %v", err))
	}
	if buf[0] != 4 {
		return proxy.Error(fmt.Errorf("This request is not a socks4 protocol"))
	}

	req := Socks4Req{}
	req.VN = buf[0]
	req.CD = buf[1]
	req.DSTPORT = uint16(buf[2])<<8 | uint16(buf[3])
	req.DSTIP = buf[4:8]
	if req.USERID, err = utils.ReadZeroString(c); err != nil {
		return proxy.Error(fmt.Errorf("socks4 reading USERID exception: %v", err))
	}
	req.HOSTNAME = net.IP(req.DSTIP).String()

	// Socks4a
	if buf[4] == 0 && buf[5] == 0 && buf[6] == 0 && buf[7] != 0 {
		if req.HOSTNAME, err = utils.ReadZeroString(c); err != nil {
			return proxy.Error(fmt.Errorf("socks4a reading HOSTNAME exception: %v", err))
		}
	}

	log.Debug("socks4 ---> VN:", req.VN,
		"; CD:", req.CD,
		"; DSTPORT:", req.DSTPORT,
		"; DSTIP:", req.DSTIP,
		"; USERID:", req.USERID,
		"; HOSTNAME:", req.HOSTNAME)

	// 验证身份
	Authenticate, err := Socks4Authenticate(req.USERID)
	if Authenticate != 90 {
		buf[0] = 0
		buf[1] = Authenticate
		c.Write(buf[:])
		return proxy.Error(err)
	}

	var rep *Socks4Rep
	switch req.CD {
	case 1: // CONNECT
		rep, err = Socks4Connect(&proxy, &req)
	case 2: // BIND
		rep, err = Socks4Bind(&proxy, &req)
	default:
		err = fmt.Errorf("socks4 does not support CD: %d", req.CD)
	}

	if err != nil {
		buf[0] = 0
		buf[1] = 91
		c.Write(buf[:])
		return proxy.Error(err)
	} else {
		repb := Socks4MakeRepBuf(rep)
		log.Trace("socks4 return data:", repb)
		c.Write(repb)
		return &proxy
	}
}

func Socks4Connect(proxy *Proxy, req *Socks4Req) (*Socks4Rep, error) {
	if targetConn, err := TryConnectServer(req.HOSTNAME, req.DSTPORT); err != nil {
		return Socks4MakeRep(proxy, req), err
	} else if targetConn != nil {
		proxy.ServerConn = targetConn
		return Socks4MakeRep(proxy, req), nil
	}

	addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", req.HOSTNAME, req.DSTPORT))
	if err != nil {
		return nil, err
	}
	targetConn, err := net.DialTCP("tcp4", nil, addr)
	if err != nil {
		return nil, err
	}
	proxy.ServerConn = targetConn

	return Socks4MakeRep(proxy, req), nil
}

func Socks4Bind(proxy *Proxy, req *Socks4Req) (*Socks4Rep, error) {
	addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", req.HOSTNAME, req.DSTPORT))
	if err != nil {
		return nil, err
	}
	listen, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		return nil, err
	}

	targetConn, err := listen.AcceptTCP()
	listen.Close()
	if err != nil {
		return nil, err
	}
	proxy.ServerConn = targetConn

	return Socks4MakeRep(proxy, req), nil
}

func Socks4MakeRep(proxy *Proxy, req *Socks4Req) *Socks4Rep {
	rep := Socks4Rep{VN: 0, CD: 90}

	remoteAddr := proxy.ServerConn.RemoteAddr().String()
	ip, portStr, _ := net.SplitHostPort(remoteAddr)
	rep.DSTIP = net.ParseIP(ip).To4()
	port, _ := strconv.Atoi(portStr)
	rep.DSTPORT = uint16(port)

	proxy.IP = rep.DSTIP
	proxy.Port = rep.DSTPORT
	proxy.HOSTNAME = req.HOSTNAME
	return &rep
}

func Socks4MakeRepBuf(rep *Socks4Rep) []byte {
	var buf [8]byte
	buf[0] = rep.VN
	buf[1] = rep.CD
	buf[2] = byte(rep.DSTPORT >> 8)
	buf[3] = byte(rep.DSTPORT)
	copy(buf[4:], rep.DSTIP)
	return buf[:]
}

func Socks4Authenticate(userId string) (byte, error) {
	// 90 成功
	// 92 无法连接到验证身份的进程
	// 93 验证身份失败
	return 90, nil
}
