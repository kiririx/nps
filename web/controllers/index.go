package controllers

import (
	"github.com/cnlh/easyProxy/server"
	"github.com/cnlh/easyProxy/utils"
)

type IndexController struct {
	BaseController
}

func (s *IndexController) Index() {
	s.SetInfo("使用说明")
	s.display("index/index")
}

func (s *IndexController) Tcp() {
	s.SetInfo("tcp隧道管理")
	s.SetType("tunnelServer")
	s.display("index/list")
}

func (s *IndexController) Udp() {
	s.SetInfo("udp隧道管理")
	s.SetType("udpServer")
	s.display("index/list")
}

func (s *IndexController) Socks5() {
	s.SetInfo("socks5管理")
	s.SetType("socks5Server")
	s.display("index/list")
}

func (s *IndexController) Http() {
	s.SetInfo("http代理管理")
	s.SetType("httpProxyServer")
	s.display("index/list")
}

func (s *IndexController) Host() {
	s.SetInfo("host模式管理")
	s.SetType("hostServer")
	s.display("index/list")
}

func (s *IndexController) GetServerConfig() {
	start, length := s.GetAjaxParams()
	taskType := s.GetString("type")
	list, cnt := server.GetServerConfig(start, length, taskType)
	s.AjaxTable(list, cnt, cnt)
}

func (s *IndexController) Add() {
	if s.Ctx.Request.Method == "GET" {
		s.Data["type"] = s.GetString("type")
		s.SetInfo("新增")
		s.display()
	} else {
		t := &server.ServerConfig{
			TcpPort:   s.GetIntNoErr("port"),
			Mode:      s.GetString("type"),
			Target:    s.GetString("target"),
			VerifyKey: utils.GetRandomString(16),
			U:         s.GetString("u"),
			P:         s.GetString("p"),
			Compress:  s.GetString("compress"),
			Crypt:     utils.GetBoolByStr(s.GetString("crypt")),
			Mux:       utils.GetBoolByStr(s.GetString("mux")),
			IsRun:     0,
		}
		server.CsvDb.NewTask(t)
		if err := server.AddTask(t); err != nil {
			s.AjaxErr(err.Error())
		} else {
			s.AjaxOk("添加成功")
		}
	}
}

func (s *IndexController) Edit() {
	if s.Ctx.Request.Method == "GET" {
		vKey := s.GetString("vKey")
		if t, err := server.CsvDb.GetTask(vKey); err != nil {
			s.error()
		} else {
			s.Data["t"] = t
		}
		s.SetInfo("修改")
		s.display()
	} else {
		vKey := s.GetString("vKey")
		if t, err := server.CsvDb.GetTask(vKey); err != nil {
			s.error()
		} else {
			t.TcpPort = s.GetIntNoErr("port")
			t.Mode = s.GetString("type")
			t.Target = s.GetString("target")
			t.U = s.GetString("u")
			t.P = s.GetString("p")
			t.Compress = s.GetString("compress")
			t.Crypt = utils.GetBoolByStr(s.GetString("crypt"))
			t.Mux = utils.GetBoolByStr(s.GetString("mux"))
			server.CsvDb.UpdateTask(t)
			server.StopServer(t.VerifyKey)
			server.StartTask(t.VerifyKey)
		}
		s.AjaxOk("修改成功")
	}
}

func (s *IndexController) Stop() {
	vKey := s.GetString("vKey")
	if err := server.StopServer(vKey); err != nil {
		s.AjaxErr("停止失败")
	}
	s.AjaxOk("停止成功")
}
func (s *IndexController) Del() {
	vKey := s.GetString("vKey")
	if err := server.DelTask(vKey); err != nil {
		s.AjaxErr("删除失败")
	}
	s.AjaxOk("删除成功")
}

func (s *IndexController) Start() {
	vKey := s.GetString("vKey")
	if err := server.StartTask(vKey); err != nil {
		s.AjaxErr("开启失败")
	}
	s.AjaxOk("开启成功")
}

func (s *IndexController) HostList() {
	if s.Ctx.Request.Method == "GET" {
		s.Data["vkey"] = s.GetString("vkey")
		s.SetInfo("域名列表")
		s.display("index/hlist")
	} else {
		start, length := s.GetAjaxParams()
		vkey := s.GetString("vkey")
		list, cnt := server.CsvDb.GetHostList(start, length, vkey)
		s.AjaxTable(list, cnt, cnt)
	}
}

func (s *IndexController) DelHost() {
	host := s.GetString("host")
	if err := server.CsvDb.DelHost(host); err != nil {
		s.AjaxErr("删除失败")
	}
	s.AjaxOk("删除成功")
}

func (s *IndexController) AddHost() {
	if s.Ctx.Request.Method == "GET" {
		s.Data["vkey"] = s.GetString("vkey")
		s.SetInfo("新增")
		s.display("index/hadd")
	} else {
		h := &server.HostList{
			Vkey:         s.GetString("vkey"),
			Host:         s.GetString("host"),
			Target:       s.GetString("target"),
			HeaderChange: s.GetString("header"),
			HostChange:   s.GetString("hostchange"),
		}
		server.CsvDb.NewHost(h)
		s.AjaxOk("添加成功")
	}
}

func (s *IndexController) EditHost() {
	if s.Ctx.Request.Method == "GET" {
		host := s.GetString("host")
		if h, t, err := server.GetKeyByHost(host); err != nil {
			s.error()
		} else {
			s.Data["t"] = t
			s.Data["h"] = h
		}
		s.SetInfo("修改")
		s.display("index/hedit")
	} else {
		host := s.GetString("host")
		if h, _, err := server.GetKeyByHost(host); err != nil {
			s.error()
		} else {
			h.Vkey = s.GetString("vkey")
			h.Host = s.GetString("nhost")
			h.Target = s.GetString("target")
			h.HeaderChange = s.GetString("header")
			h.HostChange = s.GetString("hostchange")
			server.CsvDb.UpdateHost(h)
		}
		s.AjaxOk("修改成功")
	}
}
