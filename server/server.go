package server

import (
	"errors"
	"github.com/cnlh/nps/bridge"
	"github.com/cnlh/nps/lib/file"
	"github.com/cnlh/nps/lib/lg"
	"reflect"
	"strings"
)

var (
	Bridge      *bridge.Bridge
	RunList     map[int]interface{} //运行中的任务
)

func init() {
	RunList = make(map[int]interface{})
}

//从csv文件中恢复任务
func InitFromCsv() {
	for _, v := range file.GetCsvDb().Tasks {
		if v.Status {
			lg.Println("启动模式：", v.Mode, "监听端口：", v.TcpPort)
			AddTask(v)
		}
	}
}

//start a new server
func StartNewServer(bridgePort int, cnf *file.Tunnel, bridgeType string) {
	Bridge = bridge.NewTunnel(bridgePort, RunList, bridgeType)
	if err := Bridge.StartTunnel(); err != nil {
		lg.Fatalln("服务端开启失败", err)
	}
	if svr := NewMode(Bridge, cnf); svr != nil {
		RunList[cnf.Id] = svr
		err := reflect.ValueOf(svr).MethodByName("Start").Call(nil)[0]
		if err.Interface() != nil {
			lg.Fatalln(err)
		}
	} else {
		lg.Fatalln("启动模式不正确")
	}
}

//new a server by mode name
func NewMode(Bridge *bridge.Bridge, c *file.Tunnel) interface{} {
	switch c.Mode {
	case "tunnelServer":
		return NewTunnelModeServer(ProcessTunnel, Bridge, c)
	case "socks5Server":
		return NewSock5ModeServer(Bridge, c)
	case "httpProxyServer":
		return NewTunnelModeServer(ProcessHttp, Bridge, c)
	case "udpServer":
		return NewUdpModeServer(Bridge, c)
	case "webServer":
		InitFromCsv()
		t := &file.Tunnel{
			TcpPort: 0,
			Mode:    "httpHostServer",
			Target:  "",
			Config:  &file.Config{},
			Status:  true,
		}
		AddTask(t)
		return NewWebServer(Bridge)
	case "httpHostServer":
		return NewHttp(Bridge, c)
	}
	return nil
}

//stop server
func StopServer(id int) error {
	if v, ok := RunList[id]; ok {
		reflect.ValueOf(v).MethodByName("Close").Call(nil)
		if t, err := file.GetCsvDb().GetTask(id); err != nil {
			return err
		} else {
			t.Status = false
			file.GetCsvDb().UpdateTask(t)
		}
		return nil
	}
	return errors.New("未在运行中")
}

//add task
func AddTask(t *file.Tunnel) error {
	if svr := NewMode(Bridge, t); svr != nil {
		RunList[t.Id] = svr
		go func() {
			err := reflect.ValueOf(svr).MethodByName("Start").Call(nil)[0]
			if err.Interface() != nil {
				lg.Fatalln("客户端", t.Id, "启动失败，错误：", err)
				delete(RunList, t.Id)
			}
		}()
	} else {
		return errors.New("启动模式不正确")
	}
	return nil
}

//start task
func StartTask(id int) error {
	if t, err := file.GetCsvDb().GetTask(id); err != nil {
		return err
	} else {
		AddTask(t)
		t.Status = true
		file.GetCsvDb().UpdateTask(t)
	}
	return nil
}

//delete task
func DelTask(id int) error {
	if err := StopServer(id); err != nil {
		return err
	}
	return file.GetCsvDb().DelTask(id)
}

//get key by host from x
func GetInfoByHost(host string) (h *file.Host, err error) {
	for _, v := range file.GetCsvDb().Hosts {
		s := strings.Split(host, ":")
		if s[0] == v.Host {
			h = v
			return
		}
	}
	err = errors.New("未找到host对应的内网目标")
	return
}

//get task list by page num
func GetTunnel(start, length int, typeVal string, clientId int) ([]*file.Tunnel, int) {
	list := make([]*file.Tunnel, 0)
	var cnt int
	for _, v := range file.GetCsvDb().Tasks {
		if (typeVal != "" && v.Mode != typeVal) || (typeVal == "" && clientId != v.Client.Id) {
			continue
		}
		cnt++
		if _, ok := Bridge.Client[v.Client.Id]; ok {
			v.Client.IsConnect = true
		} else {
			v.Client.IsConnect = false
		}
		if start--; start < 0 {
			if length--; length > 0 {
				if _, ok := RunList[v.Id]; ok {
					v.Client.Status = true
				} else {
					v.Client.Status = false
				}
				list = append(list, v)
			}
		}
	}
	return list, cnt
}

//获取客户端列表
func GetClientList(start, length int) (list []*file.Client, cnt int) {
	list, cnt = file.GetCsvDb().GetClientList(start, length)
	dealClientData(list)
	return
}

func dealClientData(list []*file.Client) {
	for _, v := range list {
		if _, ok := Bridge.Client[v.Id]; ok {
			v.IsConnect = true
		} else {
			v.IsConnect = false
		}
		v.Flow.InletFlow = 0
		v.Flow.ExportFlow = 0
		for _, h := range file.GetCsvDb().Hosts {
			if h.Client.Id == v.Id {
				v.Flow.InletFlow += h.Flow.InletFlow
				v.Flow.ExportFlow += h.Flow.ExportFlow
			}
		}
		for _, t := range file.GetCsvDb().Tasks {
			if t.Client.Id == v.Id {
				v.Flow.InletFlow += t.Flow.InletFlow
				v.Flow.ExportFlow += t.Flow.ExportFlow
			}
		}
	}
	return
}

//根据客户端id删除其所属的所有隧道和域名
func DelTunnelAndHostByClientId(clientId int) {
	for _, v := range file.GetCsvDb().Tasks {
		if v.Client.Id == clientId {
			DelTask(v.Id)
		}
	}
	for _, v := range file.GetCsvDb().Hosts {
		if v.Client.Id == clientId {
			file.GetCsvDb().DelHost(v.Host)
		}
	}
}

//关闭客户端连接
func DelClientConnect(clientId int) {
	Bridge.DelClient(clientId)
}

func GetDashboardData() map[string]int {
	data := make(map[string]int)
	data["hostCount"] = len(file.GetCsvDb().Hosts)
	data["clientCount"] = len(file.GetCsvDb().Clients)
	list := file.GetCsvDb().Clients
	dealClientData(list)
	c := 0
	var in, out int64
	for _, v := range list {
		if v.IsConnect {
			c += 1
		}
		in += v.Flow.InletFlow
		out += v.Flow.ExportFlow
	}
	data["clientOnlineCount"] = c
	data["inletFlowCount"] = int(in)
	data["exportFlowCount"] = int(out)
	for _, v := range file.GetCsvDb().Tasks {
		switch v.Mode {
		case "tunnelServer":
			data["tunnelServerCount"] += 1
		case "socks5Server":
			data["socks5ServerCount"] += 1
		case "httpProxyServer":
			data["httpProxyServerCount"] += 1
		case "udpServer":
			data["udpServerCount"] += 1
		}
	}
	return data
}
