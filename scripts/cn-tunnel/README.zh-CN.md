# new-api cn 网站与 API 分流隧道

版本：1.0.0。服务端 Ubuntu 22.04/24.04，客户端 Windows 10/11 或 Ubuntu 22.04/24.04（systemd）。

客户端通过 WireGuard 访问服务器的 HTTPS 服务。网站地址仍是 `https://cn.meta-api.vip/`，OpenAI 兼容 API Base URL 仍是 `https://cn.meta-api.vip/v1`，登录和 API Key 不变。只添加隧道 IP 的 `/32` 路由和该域名的 hosts 映射，不改变其他网站的默认路由。HTTP、SSE 流式响应、WebSocket 都通过原有 HTTPS 连接传输，不增加应用层缓冲或超时。

## 1. cn Ubuntu 服务器

在**实际运行该域名 HTTPS 入口的服务器**解压发行包，然后执行（把 IP 换成该服务器公网 IPv4）：

```bash
sudo bash server-install.sh --endpoint 45.136.14.237 --client-name win-office
```

这里的 IP 只是本次站点提供的入口示例；如果安装的是另一台机器，必须改成那台机器的公网 IP。不要填写 `cn.meta-api.vip`，否则客户端 hosts 修改后会导致隧道入口解析循环。

前提：

- 现有 Nginx/HTTPS 入口在宿主机监听 `0.0.0.0:443`，拥有 `cn.meta-api.vip` 有效证书，能从本机访问 `/api/status`。如果只有 new-api 的 3000 端口，需先配置 HTTPS 反向代理。脚本不创建或替换已有 Nginx 配置。
- 云安全组放行 **UDP 51820**。脚本添加本机 iptables 规则，但不能修改云安全组。已有原生 nftables/firewalld 或外部防火墙还需放行。
- `10.66.0.0/24` 不与双方现有局域网、Docker、VPN 网段冲突；冲突时使用 `--server-cidr 10.77.0.1/24 --server-ip 10.77.0.1`。
- 服务器能通过 apt 安装 WireGuard、curl、zip、iptables；需要 Python 3。客户端到服务器 UDP 端口必须可达。整个 IP 或 UDP 路径不通时，这个脚本不能保证连通。
- 隧道禁止转发到其他网络，仅开放宿主机隧道地址的 TCP 443。**Docker 直接发布 443、需要 DNAT/FORWARD 的入口不适用**；请在宿主机用 Nginx 监听 443，再反代 Docker 中的 new-api。

运行后输出客户端 ZIP 路径和 SHA256，通常为：

```text
~/wireguard-bundles/win-office.zip
```

使用 sudo 时输出在调用用户家目录；root 运行则在 `/root/wireguard-bundles/`。可以用 `--output-dir /绝对路径` 指定位置。

为另一台 Ubuntu 或 Windows 设备生成独立包：

```bash
sudo bash server-install.sh --endpoint 45.136.14.237 --add-client --client-name ubuntu-api
```

服务端密钥和旧设备配置保留，新设备自动分配空闲地址。一个 ZIP 只能给一台设备使用。重复设备名、重复地址、已有安装包会报错，不覆盖。配置包含私钥，须私下传输，不放入公开 Release 或 Git。

## 2. Windows 客户端

解压服务器生成的 ZIP，双击 **install-windows.cmd**，同意管理员权限提示即可。

也可以在解压目录执行：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\install-client.ps1
```

脚本从 WireGuard 官方站点下载安装程序，验证数字签名及发布者，安装隧道系统服务并配置 hosts。隧道开机启动，不依赖一直打开 PowerShell。如果无法下载官方安装包，先通过可信渠道安装官方 WireGuard，再运行脚本。

## 3. Ubuntu 客户端

解压对应设备 ZIP 后执行：

```bash
sudo bash install-client-linux.sh
```

## 4. 网站和接口验证

访问 `https://cn.meta-api.vip/`。Windows PowerShell 执行：

```powershell
curl.exe --noproxy "*" https://cn.meta-api.vip/api/status
# 在环境变量 NEWAPI_API_KEY 中设置原有 API Key 后：
curl.exe --noproxy "*" https://cn.meta-api.vip/v1/models -H "Authorization: Bearer $env:NEWAPI_API_KEY"
```

Ubuntu 执行：

```bash
curl --noproxy '*' https://cn.meta-api.vip/api/status
# 在环境变量 NEWAPI_API_KEY 中设置原有 API Key 后：
curl --noproxy '*' https://cn.meta-api.vip/v1/models -H "Authorization: Bearer $NEWAPI_API_KEY"
```

SDK 继续使用原域名和 API Key，流式请求继续使用 `stream: true`。网络隧道不会解除模型权限、余额或 new-api/上游已有的超时限制。

宿主机程序使用系统解析时生效。**Docker、WSL、远程执行器、浏览器独立 DNS 和显式 HTTP/SOCKS 代理不保证继承宿主机 hosts/路由**；先关闭代理做验证，容器或其他执行环境需另外配置。OAuth 回调以外的其他域名不会自动走此隧道。

服务端检查：

```bash
sudo wg show wg-newapi
sudo systemctl status wg-quick@wg-newapi --no-pager
```

`latest handshake` 及收发计数用于验证实际隧道连通。安装器检查 `/api/status`，不会使用你的 API Key，也不实际消耗模型额度。

## 5. 卸载与失败恢复

客户端连通性检查失败时会返回非零退出码，但保留隧道服务和 hosts，方便修复安全组后重试。要恢复原先直连，执行卸载脚本；**只停用 WireGuard 而不清除 hosts，会使站点仍然指向隧道地址**。

Windows：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\uninstall-client.ps1
```

Ubuntu：

```bash
sudo bash uninstall-client-linux.sh
```

卸载只删除本设备隧道配置及脚本管理的 hosts 块，保留 WireGuard 软件和其他隧道。安装成功后妥善保留卸载脚本，并删除收到的 ZIP 和配置文件副本。服务端生成包及备份同样包含密钥，应限制访问并按需清理。

停止所有此隧道客户访问（其他服务不受影响）：

```bash
sudo systemctl disable --now wg-quick@wg-newapi
```

删除单个设备需同时从 `/etc/wireguard/wg-newapi.conf` 移除该设备整个 `[Peer]` 段，并用 `sudo wg set wg-newapi peer <该设备公钥> remove` 撤销运行态访问。不要删除服务端 `[Interface]` 或其他设备段。

## 实现与验证范围

使用 [WireGuard 官方配置](https://www.wireguard.com/quickstart/)及 [Windows 隧道服务](https://git.zx2c4.com/wireguard-windows/about/docs/enterprise.md)。每个设备独立密钥和预共享密钥，Keepalive 为 25 秒，MTU 为 1280。

测试包含安装包生成、追加设备、地址冲突和 HTTPS 失败保护；CI 还进行 PowerShell 语法检查和 Ubuntu 网络命名空间内的真实 WireGuard HTTPS/API/SSE 测试。这些不代表已在福州移动或你的生产服务器实测。
