package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Conn struct {
	conn net.Conn
}

func NewConn(conn net.Conn) *Conn {
	c := new(Conn)
	c.conn = conn
	return c
}

//读取指定内容长度
func (s *Conn) ReadLen(len int) ([]byte, error) {
	raw := make([]byte, 0)
	buff := make([]byte, 1024)
	c := 0
	for {
		clen, err := s.conn.Read(buff)
		if err != nil && err != io.EOF {
			return raw, err
		}
		raw = append(raw, buff[:clen]...)
		if c += clen; c >= len {
			break
		}
	}
	if c != len {
		return raw, errors.New(fmt.Sprintf("已读取长度错误，已读取%dbyte，需要读取%dbyte。", c, len))
	}
	return raw, nil
}

//获取长度
func (s *Conn) GetLen() (int, error) {
	val := make([]byte, 4)
	_, err := s.conn.Read(val)
	if err != nil {
		return 0, err
	}
	nlen := binary.LittleEndian.Uint32(val)
	if nlen <= 0 {
		return 0, errors.New("数据长度错误")
	}
	return int(nlen), nil
}

//读取flag
func (s *Conn) ReadFlag() (string, error) {
	val := make([]byte, 4)
	_, err := s.conn.Read(val)
	if err != nil {
		return "", err
	}
	return string(val), err
}

//读取host
func (s *Conn) GetHostFromConn() (string, error) {
	len, err := s.GetLen()
	if err != nil {
		return "", err
	}
	hostByte := make([]byte, len)
	_, err = s.conn.Read(hostByte)
	if err != nil {
		return "", err
	}
	return string(hostByte), nil
}

//获取host
func (s *Conn) WriteHost(host string) (int, error) {
	raw := bytes.NewBuffer([]byte{})
	binary.Write(raw, binary.LittleEndian, int32(len([]byte(host))))
	binary.Write(raw, binary.LittleEndian, []byte(host))
	return s.Write(raw.Bytes())
}

//设置连接为长连接
func (s *Conn) SetAlive() {
	conn := s.conn.(*net.TCPConn)
	conn.SetReadDeadline(time.Time{})
	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(time.Duration(2 * time.Second))
}

//从tcp报文中解析出host
func (s *Conn) GetHost() (method, address string, rb []byte, err error) {
	var b [2048]byte
	var n int
	var host string
	if n, err = s.Read(b[:]); err != nil {
		return
	}
	rb = b[:n]
	//TODO：某些不规范报文可能会有问题
	fmt.Sscanf(string(b[:n]), "%s", &method)
	reg, err := regexp.Compile(`(\w+:\/\/)([^/:]+)(:\d*)?`)
	if err != nil {
		return
	}
	host = string(reg.Find(b[:]))
	hostPortURL, err := url.Parse(host)
	if err != nil {
		return
	}
	if hostPortURL.Opaque == "443" { //https访问
		address = hostPortURL.Scheme + ":443"
	} else { //http访问
		if strings.Index(hostPortURL.Host, ":") == -1 { //host不带端口， 默认80
			address = hostPortURL.Host + ":80"
		} else {
			address = hostPortURL.Host
		}
	}
	return
}

func (s *Conn) Close() error {
	return s.conn.Close()
}
func (s *Conn) Write(b []byte) (int, error) {
	return s.conn.Write(b)
}
func (s *Conn) Read(b []byte) (int, error) {
	return s.conn.Read(b)
}

func (s *Conn) wError() {
	s.conn.Write([]byte(RES_MSG))
}

func (s *Conn) wMain() {
	s.conn.Write([]byte(WORK_MAIN))
}

func (s *Conn) wChan() {
	s.conn.Write([]byte(WORK_CHAN))
}
