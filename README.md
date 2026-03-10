# netokeep

Setup SSH tunnels and HTTP proxies to keep your restricted containers connected to the outside world.

在受限容器中设置 SSH 隧道和 HTTP 代理，以保持它们与外界的连接。

```text
# Use Case
Container Listening (No Internet) <--> HTTP/HTTPS Link <--> Users
```

## Quick start

### Server Part

Setup the netokeep server
> Please ensure that your SSH service has started to the input port for SSH connection.

```bash
nks start -i 22  -o 7222
```
Then get the HTTP Link provided by your company <HTTP_LINK>.

### Client Part

Create connection to you server

```bash
nk start -r <HTTP_LINK> -o 2222
```

And enjoy!
