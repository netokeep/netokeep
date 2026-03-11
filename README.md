# netokeep

Setup SSH tunnels and TCP proxies to keep your restricted containers connected to the outside world.

在受限容器中设置 SSH 隧道和 TCP 代理，以保持它们与外界的连接。

```text
# Use Case
Container Listening (No Internet) <--> HTTP/HTTPS Link <--> Users
```

> Currently, netokeep only supports ws connection.
>
> 目前 netokeep 仅支持 ws 连接。

## Download
You can download the latest release of NetoKeep and install it. It contains both the server and client parts.

下载并安装最新版本的 NetoKeep，它包含服务器和客户端两部分。

```bash
# Linux AMD64
wget github.com/netokeep/netokeep/releases/latest/download/netokeep-Linux-amd64.sh
chmod +x netokeep-Linux-amd64.sh
sudo ./netokeep-Linux-amd64.sh
rm -f netokeep-Linux-amd64.sh
```

## Quick start

### Server Part

Setup the NetoKeep server

> Please ensure your SSH service is running and accessible on the given port.

```bash
nks start -s 22 -t 1080 -o 7222
```
Then get the HTTP Link for port 7222 provided by your company <HTTP_LINK>.

### Client Part

Create the NetoKeep client to connect to your server

```bash
nk start -s 2222 -r <HTTP_LINK>
```

And enjoy!

## Acknowledgement

- [rtunnel](https://github.com/Sarfflow/rtunnel)
- [tcp-over-websocket](https://github.com/zanjie1999/tcp-over-websocket)
- [yamux](https://github.com/hashicorp/yamux)
