<div align="center">

<h1>NetoKeep</h1>

[![go](https://img.shields.io/github/go-mod/go-version/netokeep/netokeep)](https://golang.org/)
[![license](https://img.shields.io/github/license/netokeep/netokeep)](LICENSE)
[![release](https://img.shields.io/github/v/release/netokeep/netokeep?color=green)](https://github.com/netokeep/netokeep/releases)
![downloads](https://img.shields.io/github/downloads/netokeep/netokeep/total)

</div>

在受限容器中设置 SSH 隧道和 TCP 代理，以保持它们与外界的连接。

适用场景：`容器（无外部网络）<--> 代理的HTTP/HTTPS连接 <--> 用户设备`

## 安装

**在线安装**

在`MacOS, Linux`上下载：
```bash
curl -fsSL https://raw.githubcontent.com/netokeep/netokeep/main/scripts/install.sh | bash
```
在`Windows`上下载：

```bash
irm https://raw.githubcontent.com/netokeep/netokeep/main/scripts/install.ps1 | iex
```

**离线安装**

同时下载对应的软件包和安装脚本`scripts/install-offline.sh`或`scripts/install-offline.ps1`。

将两者放在统一目录下，并执行对应的离线安装脚本。

## 快速开始

NetoKeep包含服务器和客户端两部分。其中服务器在容器中运行，客户端在用户设备上运行。

### NetoKeep Server

```bash
nks start
```
> - i：流量代理端口（默认关闭）
> - o；前向转发端口（默认为7222）
> - 流量转发的配置文件位置`~/.config/netokeep/nks_settings.json`


### NetoKeep Client

```bash
nk start <name> -r <remote-link> -s <ssh-port>
```
> - r：远程提供的代理HTTP/HTTPS连接
> - s：暴露给本机用于ssh连接的端口
> - f：将接收到的流量进一步转发（默认关闭）
> - 流量转发的配置文件位置`~/.config/netokeep/nks_settings.json`

### 日常使用

```bash
ssh -p <ssh-port> root@localhost
```

Enjoy!

## 致谢

- [rtunnel](https://github.com/Sarfflow/rtunnel)
- [tcp-over-websocket](https://github.com/zanjie1999/tcp-over-websocket)
- [yamux](https://github.com/hashicorp/yamux)
