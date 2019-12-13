
# nps
![](https://img.shields.io/github/stars/cnlh/nps.svg)   ![](https://img.shields.io/github/forks/cnlh/nps.svg)
[![Gitter](https://badges.gitter.im/cnlh-nps/community.svg)](https://gitter.im/cnlh-nps/community?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)
[![Build Status](https://travis-ci.org/cnlh/nps.svg?branch=master)](https://travis-ci.org/cnlh/nps)

nps是一款轻量级、高性能、功能强大的**内网穿透**代理服务器。目前支持**tcp、udp流量转发**，可支持任何**tcp、udp**上层协议（访问内网网站、本地支付接口调试、ssh访问、远程桌面，内网dns解析等等……），此外还**支持内网http代理、内网socks5代理**、**p2p等**，并带有功能强大的web管理端。


## 背景
![image](https://github.com/cnlh/nps/blob/master/image/web.png?raw=true)

1. 做微信公众号开发、小程序开发等----> 域名代理模式

2. 想在外网通过ssh连接内网的机器，做云服务器到内网服务器端口的映射，----> tcp代理模式

3. 在非内网环境下使用内网dns，或者需要通过udp访问内网机器等----> udp代理模式

4. 在外网使用HTTP代理访问内网站点----> http代理模式

5. 搭建一个内网穿透ss，在外网如同使用内网vpn一样访问内网资源或者设备----> socks5代理模式

## 快速开始

### 安装
> [releases](https://github.com/cnlh/nps/releases)

下载对应的系统版本即可，服务端和客户端是单独的

### 服务端启动
1. 进入服务端启动
```shell
 ./nps
```
如有错误修改配置文件相应端口，无错误可继续进行下去

2. 访问服务端ip:web服务端口（默认为8024）
3. 使用用户名和密码登陆（默认admin/123，正式使用一定要更改）
4. 在web中创建客户端

### 客户端连接
1. 点击web管理中客户端前的+号，复制启动命令
2. 执行启动命令，linux直接执行即可，windows将./npc换成npc.exe用cmd执行

### 配置
- 客户端连接后，在web中配置对应穿透服务即可
- 更多高级用法见[完整文档](https://cnlh.github.io/nps/)

## 贡献
- 如果遇到bug可以直接提交至dev分支
- 使用遇到问题可以通过issues反馈
- 项目处于开发阶段，还有很多待完善的地方，如果可以贡献代码，请提交 PR 至 dev 分支
- 如果有新的功能特性反馈，可以通过issues或者qq群反馈
