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
wget github.com/netokeep/netokeep/releases/latest/download/netokeep-linux-amd64.sh
chmod +x netokeep-linux-amd64.sh
./netokeep-linux-amd64.sh
rm -f netokeep-linux-amd64.sh
```

## Quick start

### Server Part
#### 1. Install dependencies

```bash
# Add your pubKey into ssh config 添加你的客户端公钥到ssh配置文件中
read -r -p "Input SSH public key (Enter to skip): " K
[ -n "$K" ] && install -d -m 700 ~/.ssh && touch ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys
[ -n "$K" ] && (grep -qxF "$K" ~/.ssh/authorized_keys 2>/dev/null || echo "$K" >> ~/.ssh/authorized_keys)
# Install tools 安装工具
apt update -y
apt install -y sudo openssh-server tmux
ssh-keygen -A
```

#### 2. Start SSH service
```bash
sudo -v && \
tmux has-session -t sshd 2>/dev/null || \
tmux new-session -d -s sshd \
  "sudo mkdir -p /run/sshd && sudo /usr/sbin/sshd -D -e"
```

#### 3. Setup the NetoKeep server
Open a new terminal and run:
```bash
# s: The port for your SSH service
# t: The TCP proxy port (protocol: HTTP)
# o: The forwarding port for SSH traffc and Proxy traffic
nks start -s 22 -t 7890 -o 7222
```
Then get the HTTP Link for port 7222 provided by your company <HTTP_LINK>.

### Client Part

#### 1. Setup the NetoKeep client

Create the NetoKeep client to connect to your server

```bash
# f: forward the server traffic (default false)
nk start -f -s 2222 -r <HTTP_LINK>
```

#### 2. Connect to your container using SSH

```bash
# ssh-keygen -R "[localhost]:2222"
ssh -p 2222 root@localhost
```
> If you want to enable Internet access for your container,
> run the following command after connecting to your container.
> (You can also write them into your `.bashrc` file.)
> ```bash
> export ALL_PROXY=http://127.0.0.1:7890
> export HTTP_PROXY=http://127.0.0.1:7890
> export HTTPS_PROXY=http://127.0.0.1:7890
> ```

And enjoy!

> [!TIP]
> If your container cannot download the VS Code Server, you can add the following to your VS Code settings to use the local proxy for downloading:
> ```json
> {
>	"remote.SSH.localServerDownload": "always",
> }
> ```

## Acknowledgement

- [rtunnel](https://github.com/Sarfflow/rtunnel)
- [tcp-over-websocket](https://github.com/zanjie1999/tcp-over-websocket)
- [yamux](https://github.com/hashicorp/yamux)
