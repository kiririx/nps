<<<<<<< Updated upstream
# easyProxy
轻量级、较高性能http代理服务器，主要应用与内网穿透。支持多站点配置、客户端与服务端连接中断自动重连，多路传输，大大的提高请求处理速度，go语言编写，无第三方依赖，经过测试内存占用小，普通场景下，仅占用10m内存。

## 背景	  
我有一个小程序的需求，但是小程序的数据源必须从内网才能抓取到，但是又苦于内网服务器没有公网ip，所以只能内网穿透了。

用了一段时间ngrok做内网穿透，可能由于功能比较强大，配置起来挺麻烦的，加之开源版有内存的泄漏，很是闹心。

正好最近在看go相关的东西，所以做了一款代理服务器，功能比较简单，用于内网穿透最为合适。

## 特点
- [x] 支持gzip压缩,减小流量消耗
- [x] 支持多站点配置
- [x] 断线自动重连
- [x] 支持多路传输,提高并发
- [x] 跨站自动匹配替换
## 安装
1. release安装
> https://github.com/cnlh/easyProxy/releases

下载对应的系统版本即可（目前linux和windows只编译了64位的），服务端和客户端共用一个程序，go语言开发，无需任何第三方依赖

2. 源码安装
- 安装源码
> go get github.com/cnlh/easyProxy
- 编译（无第三方模块）
> go build

## 使用 
- 服务端 

```
./easyProxy -mode server -vkey DKibZF5TXvic1g3kY -tcpport=8284 -httpport=8024
```

名称 | 含义
---|---
mode | 运行模式(client、server不写默认client)
vkey | 验证密钥
tcpport | 服务端与客户端通信端口
httpport | 代理的http端口（与nginx配合使用）

- 客户端


```
建立配置文件 config.json
```


```
./easyProxy -config config.json  
```


 名称 | 含义
---|---
config | 配置文件路径
## 配置文件config.json

```
{
  "Server": {
    "ip": "123.206.77.88",
    "tcp": 8284,
    "vkey": "DKibZF5TXvic1g3kY",
    "num": 10
  },
  "SiteList": [
    {
      "host": "server1.ourcauc.com",
      "url": "10.1.50.203",
      "port": 80
    },
    {
      "host": "server2.ourcauc.com",
      "url": "10.1.50.196",
      "port": 4000
    }
  ],
  "Replace": 0
}
```
 名称 | 含义
---|---
ip | 服务端ip地址
tcp | 服务端与客户端通信端口
vkey | 验证密钥
num | 服务端与客户端通信连接数
SiteList | 本地解析的域名列表
host | 域名地址
url | 内网代理的地址
port | 内网代理的地址对应的端口

## 运行流程解析



```
graph TD
A[通过域名访问对应内网服务]-->B[nginx代理转发该域名服务端监听的8024端口]
B-->C[服务端将请求发送到客户端上]
C-->D[客户端收到请求信息,根据host判断对应的内网的请求地址,执行对应请求]
D-->E[将请求结果返回给服务端]
E-->F[服务端收到后返回给访问者]
```

## nginx代理配置示例
```
upstream nodejs {
    server 127.0.0.1:8024;
    keepalive 64;
}
server {
    listen 80;
    server_name server1.ourcauc.com server2.ourcauc.com;
    location / {
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header Host  $http_host:8024;
            proxy_set_header X-Nginx-Proxy true;
            proxy_set_header Connection "";
            proxy_pass      http://nodejs;
        }
}
```
## 域名配置示例
> -server1	    A	    123.206.77.88

> -server2	    A	    123.206.77.88

## 跨站自动匹配替换说明

例如，访问：server1.ourcauc.com，该页面里面有一个超链接为10.1.50.196:4000,将根据配置文件自动该将url替换为server2.ourcauc.com，以达到跨站也可访问的效果，但需要提前在配置文件中配置这些站点。

如需开启，请加配置文件Replace值设置为1
>注意：开启可能导致不应该被替换的内容被替换，请谨慎开启
=======
# rproxy
简单的反向代理用于内网穿透  

**特别注意，此工具只适合小文件类的访问测试，用来做做数据调试。当初也只是用于微信公众号开发，所以定位也是如此** 

## 前言	  
最近周末闲来无事，想起了做下微信公共号的开发，但微信限制只能80端口的，自己用的城中村的那种宽带，共用一个公网，没办法自己用路由做端口映射。自己的服务器在腾讯云上，每次都要编译完后用ftp上传再进行调试，非常的浪费时间。 一时间又不知道上哪找一个符合我的这种要求的工具，就索性自己构思了下，整个工作流程大致为：   

## 工作原理  
> 外部请求自己服务器上的HTTP服务端 -> 将数据传递给Socket服务器 -> Socket服务器将数据发送至已连接的Socket客户端 -> Socket客户端收到数据 -> 使用http请求本地http服务端 -> 本地http服务端处理相关后返回 -> Socket客户端将返回的数据发送至Socket服务端 -> Socket服务端解析出数据后原路返回至外部请求的HTTP  
 
## 使用方法  
> 1、go get github.com/ying32/rproxy  
> 2、go build   
> 3、服务端运行runsvr.bat或者runsvr.sh    
> 4、客户端运行runcli.bat或者runcli.sh    

## 命令行说明    
>  --tcpport    Socket连接或者监听的端口   
>  --httpport   当mode为server时为服务端监听端口，当为mode为client时为转发至本地客户端的端口  
>  --mode       启动模式，可选为client、server，默认为client  
>  --svraddr    当mode为client时有效，为连接服务器的地址，不需要填写端口    
>  --vkey       客户端与服务端建立连接时校验的加密key，简单的。  
>>>>>>> Stashed changes

## 操作系统支持  
支持Windows、Linux、MacOSX等，无第三方依赖库。  

## 二进制下载
https://github.com/ying32/rproxy/releases/tag/v0.4  
