# NetoKeep

[![go](https://img.shields.io/github/go-mod/go-version/netokeep/netokeep)](https://golang.org/)
[![license](https://img.shields.io/github/license/netokeep/netokeep)](LICENSE)
[![release](https://img.shields.io/github/v/release/netokeep/netokeep?color=green)](https://github.com/netokeep/netokeep/releases)
![downloads](https://img.shields.io/github/downloads/netokeep/netokeep/total)

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
# This command is for Linux AMD64.
# Download the latest release for based on your system
INSTALLER="netokeep-linux-amd64-installer"
wget "github.com/netokeep/netokeep/releases/latest/download/$INSTALLER"
chmod +x "$INSTALLER"
./"$INSTALLER"
rm -f "$INSTALLER"
```

Install dependencies.

安装依赖。

```bash
# Add your pubKey into ssh config 添加你的客户端公钥到ssh配置文件中
read -r -p "Input SSH public key (Enter to skip): " K
[ -n "$K" ] && install -d -m 700 ~/.ssh && touch ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys
[ -n "$K" ] && (grep -qxF "$K" ~/.ssh/authorized_keys 2>/dev/null || echo "$K" >> ~/.ssh/authorized_keys)
# Install tools 安装工具
apt update -y
apt install -y sudo openssh-server
ssh-keygen -A
```


## Quick start

### Server Part

Open a new terminal and run:

```bash
nks start
```
> - i: The TCP proxy port (protocol: HTTP, default 7890)
> - o: The forwarding port for SSH traffc and Proxy traffic (default 7222)
> - Control the traffic to proxy in `~/.config/netokeep/nks_settings.json`

Then get the HTTP Link for port 7222 provided by your company <HTTP_LINK>.

### Client Part

#### 1. Setup the NetoKeep client

Create the NetoKeep client to connect to your server:

```bash
nk start -r <HTTP_LINK>
```
> - s: Assign the port for ssh connection (default 2222)
> - f: Forward the server traffic (default false)
> - p: Use proxy rules in `~/.config/netokeep/nk_settings.json` (default false)

#### 2. Connect to your container using SSH

```bash
# ssh-keygen -R "[localhost]:2222" # command for removing previous host
ssh -p 2222 root@localhost
```
> If you want to enable Internet access for your container,
> run the following command after connecting to your container.
> 
> (You can also write them into your `.bashrc` file. Not recommended.)
> ```bash
> export ALL_PROXY=http://127.0.0.1:7890
> export HTTP_PROXY=http://127.0.0.1:7890
> export HTTPS_PROXY=http://127.0.0.1:7890
> ```

And enjoy!

## Notes

The previous programs are stored in differnt locations. You could run the following commands to remove them.

```bash
sudo rm -rf /usr/local/bin/nk
sudo rm -rf /usr/local/bin/nks
sudo rm -rf ~/.local/share/netokeep
```

## Acknowledgement

- [rtunnel](https://github.com/Sarfflow/rtunnel)
- [tcp-over-websocket](https://github.com/zanjie1999/tcp-over-websocket)
- [yamux](https://github.com/hashicorp/yamux)
