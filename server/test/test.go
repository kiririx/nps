package test

import (
	"github.com/cnlh/nps/lib/common"
	"github.com/cnlh/nps/lib/file"
	"github.com/cnlh/nps/vender/github.com/astaxie/beego"
	"log"
	"strconv"
)

func TestServerConfig() {
	var postTcpArr []int
	var postUdpArr []int
	for _, v := range file.GetCsvDb().Tasks {
		if v.Mode == "udp" {
			isInArr(&postUdpArr, v.Port, v.Remark, "udp")
		} else {
			isInArr(&postTcpArr, v.Port, v.Remark, "tcp")
		}
	}
	p, err := beego.AppConfig.Int("web_port")
	if err != nil {
		log.Fatalln("Getting web management port error :", err)
	} else {
		isInArr(&postTcpArr, p, "Web Management port", "tcp")
	}

	if p := beego.AppConfig.String("bridge_port"); p != "" {
		if port, err := strconv.Atoi(p); err != nil {
			log.Fatalln("get Server and client communication portserror:", err)
		} else if beego.AppConfig.String("bridge_type") == "kcp" {
			isInArr(&postUdpArr, port, "Server and client communication ports", "udp")
		} else {
			isInArr(&postTcpArr, port, "Server and client communication ports", "tcp")
		}
	}

	if p := beego.AppConfig.String("httpProxyPort"); p != "" {
		if port, err := strconv.Atoi(p); err != nil {
			log.Fatalln("get http port error:", err)
		} else {
			isInArr(&postTcpArr, port, "https port", "tcp")
		}
	}
	if p := beego.AppConfig.String("https_proxy_port"); p != "" {
		if port, err := strconv.Atoi(p); err != nil {
			log.Fatalln("get https port error", err)
		} else {
			if !common.FileExists(beego.AppConfig.String("pemPath")) {
				log.Fatalf("ssl certFile %s is not exist", beego.AppConfig.String("pemPath"))
			}
			if !common.FileExists(beego.AppConfig.String("ketPath")) {
				log.Fatalf("ssl keyFile %s is not exist", beego.AppConfig.String("pemPath"))
			}
			isInArr(&postTcpArr, port, "http port", "tcp")
		}
	}
}

func isInArr(arr *[]int, val int, remark string, tp string) {
	for _, v := range *arr {
		if v == val {
			log.Fatalf("the port %d is reused,remark: %s", val, remark)
		}
	}
	if tp == "tcp" {
		if !common.TestTcpPort(val) {
			log.Fatalf("open the %d port error ,remark: %s", val, remark)
		}
	} else {
		if !common.TestUdpPort(val) {
			log.Fatalf("open the %d port error ,remark: %s", val, remark)
		}
	}
	*arr = append(*arr, val)
	return
}
