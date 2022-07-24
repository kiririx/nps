package conn

import (
	"net"
	"strings"
	"sync"

	"github.com/astaxie/beego/logs"
	"github.com/xtaci/kcp-go"
)

var (
	IpC     = sync.Map{}
	BlackIp = make(map[string]bool)
)

func NewTcpListenerAndProcess(addr string, f func(c net.Conn), listener *net.Listener) error {
	var err error
	*listener, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	Accept(*listener, f)
	return nil
}

func NewKcpListenerAndProcess(addr string, f func(c net.Conn)) error {
	kcpListener, err := kcp.ListenWithOptions(addr, nil, 150, 3)
	if err != nil {
		logs.Error(err)
		return err
	}
	for {
		c, err := kcpListener.AcceptKCP()
		SetUdpSession(c)
		if err != nil {
			logs.Warn(err)
			continue
		}
		go f(c)
	}
	return nil
}

func Accept(l net.Listener, f func(c net.Conn)) {
	for {
		c, err := l.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}
			if strings.Contains(err.Error(), "the mux has closed") {
				break
			}
			logs.Warn(err)
			continue
		}
		// check ip in black list
		remoteAddr := c.RemoteAddr()
		ip, _, _ := strings.Cut(remoteAddr.String(), ":")
		v, ok := IpC.Load(ip)
		if !ok {
			IpC.Store(ip, int64(1))
		}
		v, _ = IpC.Load(ip)
		IpC.Store(ip, v.(int64)+1)
		if BlackIp[ip] {
			// logs.Warn("ip %s is black", ip)
			c.Close()
			continue
		}
		if c == nil {
			logs.Warn("nil connection")
			break
		}
		go f(c)
	}
}
