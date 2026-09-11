#!/usr/bin/env bash
set -Eeuo pipefail

umask 077

INTERFACE="wg-newapi"
LISTEN_PORT="51820"
SERVER_CIDR="10.66.0.1/24"
SERVER_TUNNEL_IP="10.66.0.1"
CLIENT_CIDR=""
PUBLIC_ENDPOINT=""
DOMAIN="cn.meta-api.vip"
CLIENT_NAME="newapi-fuzhou"
OUTPUT_DIR=""
ADD_CLIENT=0
CN_TUNNEL_VERSION="1.1.0"

log() {
  printf '[wireguard] %s\n' "$*"
}

die() {
  printf '[wireguard] ERROR: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'USAGE'
用法：
  sudo bash server-install.sh --endpoint <服务器公网IPv4> [选项]

必填：
  --endpoint VALUE       WireGuard 公网 IPv4 入口，例如 45.136.14.237

选项：
  --domain VALUE         客户访问域名，默认 cn.meta-api.vip
  --client-name VALUE    客户隧道名，默认 newapi-fuzhou
  --listen-port VALUE    WireGuard UDP 端口，默认 51820
  --interface VALUE      服务端接口名，默认 wg-newapi
  --server-cidr VALUE    服务端隧道地址，默认 10.66.0.1/24
  --server-ip VALUE      客户访问的隧道 IP，默认 10.66.0.1
  --client-cidr VALUE    客户端隧道地址，默认自动分配空闲地址
  --output-dir VALUE     客户安装包输出目录
  --add-client           向已有隧道新增设备（保留已有设备和密钥）
  -h, --help             显示帮助

示例：
  sudo bash server-install.sh \
    --endpoint 45.136.14.237 \
    --domain cn.meta-api.vip \
    --client-name newapi-fuzhou
USAGE
}

while [ "$#" -gt 0 ]; do
  if [[ "$1" == --* && "$1" != --help && "$1" != --add-client ]]; then
    [ "$#" -ge 2 ] && [ -n "$2" ] || die "$1 缺少参数"
  fi
  case "$1" in
    --endpoint) PUBLIC_ENDPOINT="${2:-}"; shift 2 ;;
    --domain) DOMAIN="${2:-}"; shift 2 ;;
    --client-name) CLIENT_NAME="${2:-}"; shift 2 ;;
    --listen-port) LISTEN_PORT="${2:-}"; shift 2 ;;
    --interface) INTERFACE="${2:-}"; shift 2 ;;
    --server-cidr) SERVER_CIDR="${2:-}"; shift 2 ;;
    --server-ip) SERVER_TUNNEL_IP="${2:-}"; shift 2 ;;
    --client-cidr) CLIENT_CIDR="${2:-}"; shift 2 ;;
    --output-dir) OUTPUT_DIR="${2:-}"; shift 2 ;;
    --add-client) ADD_CLIENT=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) die "未知参数：$1" ;;
  esac
done

[ "$(id -u)" -eq 0 ] || die "请使用 root 或 sudo 运行"
[ -n "$PUBLIC_ENDPOINT" ] || die "缺少 --endpoint"
[[ "$PUBLIC_ENDPOINT" =~ ^[A-Za-z0-9.-]+$ ]] || die "--endpoint 格式无效"
[[ "$DOMAIN" =~ ^[A-Za-z0-9.-]+$ ]] || die "--domain 格式无效"
[[ "$CLIENT_NAME" =~ ^[A-Za-z0-9_-]{1,15}$ ]] || die "--client-name 仅允许字母、数字、下划线、短横线，最长 15"
[[ "$INTERFACE" =~ ^[A-Za-z0-9_-]{1,15}$ ]] || die "--interface 最长 15，仅允许字母、数字、下划线、短横线"
[[ "$LISTEN_PORT" =~ ^[0-9]{1,5}$ ]] || die "--listen-port 格式无效"
[ "$LISTEN_PORT" -ge 1 ] && [ "$LISTEN_PORT" -le 65535 ] || die "端口范围无效"
[[ "$SERVER_CIDR" =~ ^[0-9.]+/[0-9]+$ ]] || die "--server-cidr 格式无效"
[[ "$SERVER_TUNNEL_IP" =~ ^[0-9.]+$ ]] || die "--server-ip 格式无效"
[ -z "$CLIENT_CIDR" ] || [[ "$CLIENT_CIDR" =~ ^[0-9.]+/32$ ]] || die "--client-cidr 必须是 IPv4 /32"
[[ "$CLIENT_NAME" =~ ^[A-Za-z0-9] ]] || die "客户端名称必须以字母或数字开头"

if [ -z "$OUTPUT_DIR" ]; then
  if [ -n "${SUDO_USER:-}" ] && [ "$SUDO_USER" != "root" ]; then
    OWNER_HOME="$(getent passwd "$SUDO_USER" | cut -d: -f6)"
  else
    OWNER_HOME="/root"
  fi
  OUTPUT_DIR="$OWNER_HOME/wireguard-bundles"
fi

[ "${OUTPUT_DIR:0:1}" = / ] || die "--output-dir 必须为绝对路径"

command -v systemctl >/dev/null 2>&1 || die "系统不支持 systemd"
command -v apt-get >/dev/null 2>&1 || die "当前脚本仅支持 Ubuntu/Debian"

command -v python3 >/dev/null 2>&1 || die "请先运行 sudo apt-get install -y python3"
exec 9>/run/lock/newapi-cn-tunnel.lock
flock -n 9 || die "另一安装进程正在运行"
CONF="/etc/wireguard/${INTERFACE}.conf"
if [ "$ADD_CLIENT" -eq 1 ]; then
  [ -f "$CONF" ] || die "尚无服务端配置，请先执行首次安装"
  grep -qx '# Managed by new-api cn-tunnel v1' "$CONF" || die "拒绝修改非本脚本创建的隧道"
  grep -qx "# ${CLIENT_NAME}" "$CONF" && die "设备名已存在，请使用新名称"
  SERVER_CIDR="$(sed -n 's/^Address = //p' "$CONF")"
  SERVER_TUNNEL_IP="${SERVER_CIDR%/*}"
  LISTEN_PORT="$(sed -n 's/^ListenPort = //p' "$CONF")"
else
  [ ! -e "$CONF" ] || die "$CONF 已存在；新增设备请使用 --add-client，脚本不会覆盖原配置"
fi
CLIENT_CIDR="$(python3 - "$SERVER_CIDR" "$SERVER_TUNNEL_IP" "$CLIENT_CIDR" "$CONF" "$PUBLIC_ENDPOINT" <<'VALIDATE'
import ipaddress, pathlib, sys
server, target, client, filename, endpoint = sys.argv[1:]
try:
    ipaddress.IPv4Address(endpoint)  # Avoid the endpoint domain being redirected by hosts.
    iface = ipaddress.IPv4Interface(server)
    assert 24 <= iface.network.prefixlen <= 30, '服务端网段前缀须为 /24 到 /30'
    assert str(iface.ip) == target, '--server-ip 必须与 --server-cidr 地址一致'
    assert iface.ip not in (iface.network.network_address, iface.network.broadcast_address)
    used = {iface.ip}
    path = pathlib.Path(filename)
    if path.exists():
        for line in path.read_text().splitlines():
            if line.startswith('AllowedIPs = '):
                used.add(ipaddress.IPv4Interface(line.split(' = ')[1]).ip)
    addr = ipaddress.IPv4Interface(client) if client else next(
        (ipaddress.IPv4Interface(str(ip) + '/32') for ip in iface.network.hosts() if ip not in used), None)
    assert addr is not None, '地址池已满'
    assert addr.network.prefixlen == 32 and addr.ip in iface.network, '客户端必须是服务端网段内的 /32'
    assert addr.ip not in used and addr.ip not in (iface.network.network_address, iface.network.broadcast_address), '客户端地址已占用或无效'
    print(addr)
except (ValueError, AssertionError) as exc:
    sys.exit('地址校验失败（endpoint 必须为公网 IPv4）：' + str(exc))
VALIDATE
)"
[ ! -e "${OUTPUT_DIR}/${CLIENT_NAME}" ] && [ ! -e "${OUTPUT_DIR}/${CLIENT_NAME}.zip" ] || die "安装包已存在，请先移走或换设备名"

log "安装 WireGuard 与打包工具"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y wireguard wireguard-tools zip ca-certificates curl iptables >/dev/null

# Existing HTTPS must listen on all IPv4 addresses, including the tunnel address.
# Check loopback first to avoid changing a server whose HTTPS is not configured.
curl --noproxy '*' -fsS --resolve "${DOMAIN}:443:127.0.0.1" --connect-timeout 5 --max-time 20 \
  "https://${DOMAIN}/api/status" >/dev/null || die "本机 443 HTTPS 检查失败。请先让现有 Nginx 监听 0.0.0.0:443 并配置该域名有效证书，再重新运行"

if [ "$ADD_CLIENT" -eq 1 ]; then
  SERVER_PRIVATE="$(sed -n 's/^PrivateKey = //p' "$CONF")"
  systemctl start "wg-quick@${INTERFACE}"
else
  SERVER_PRIVATE="$(wg genkey)"
fi
SERVER_PUBLIC="$(printf '%s' "$SERVER_PRIVATE" | wg pubkey)"
CLIENT_PRIVATE="$(wg genkey)"
CLIENT_PUBLIC="$(printf '%s' "$CLIENT_PRIVATE" | wg pubkey)"
PRESHARED_KEY="$(wg genpsk)"

install -d -m 700 /etc/wireguard
if [ "$ADD_CLIENT" -eq 0 ]; then
  cat > "$CONF" <<WGCONF
# Managed by new-api cn-tunnel v1
[Interface]
Address = ${SERVER_CIDR}
ListenPort = ${LISTEN_PORT}
PrivateKey = ${SERVER_PRIVATE}
MTU = 1280
# Only HTTPS to this host; disallow forwarding and access to other local services.
PostUp = iptables -I INPUT 1 -i %i -j DROP; iptables -I INPUT 1 -i %i -d ${SERVER_TUNNEL_IP} -p tcp --dport 443 -j ACCEPT; iptables -I FORWARD 1 -i %i -j DROP; iptables -I INPUT 1 -p udp --dport ${LISTEN_PORT} -j ACCEPT
PostDown = iptables -D INPUT -i %i -j DROP; iptables -D INPUT -i %i -d ${SERVER_TUNNEL_IP} -p tcp --dport 443 -j ACCEPT; iptables -D FORWARD -i %i -j DROP; iptables -D INPUT -p udp --dport ${LISTEN_PORT} -j ACCEPT
WGCONF
  chmod 600 "$CONF"
  systemctl enable --now "wg-quick@${INTERFACE}" >/dev/null
fi
curl --noproxy '*' -fsS --resolve "${DOMAIN}:443:${SERVER_TUNNEL_IP}" --connect-timeout 5 --max-time 20 \
  "https://${DOMAIN}/api/status" >/dev/null || die "隧道地址 HTTPS 检查失败；请检查 Nginx 监听地址。服务端配置已保留，修复后用 --add-client 重试"

# Append only after HTTPS preflight; existing peers are never replaced.
cp -a "$CONF" "${CONF}.bak-$(date +%Y%m%d%H%M%S)"
cat >> "$CONF" <<WGCONF

[Peer]
# ${CLIENT_NAME}
PublicKey = ${CLIENT_PUBLIC}
PresharedKey = ${PRESHARED_KEY}
AllowedIPs = ${CLIENT_CIDR}
WGCONF
STRIPPED="$(mktemp /etc/wireguard/.cn-sync.XXXXXX)"
trap 'rm -f "$STRIPPED"' EXIT
wg-quick strip "$CONF" > "$STRIPPED"
wg syncconf "$INTERFACE" "$STRIPPED"
rm -f "$STRIPPED"

BUNDLE_DIR="${OUTPUT_DIR}/${CLIENT_NAME}"
ZIP_PATH="${OUTPUT_DIR}/${CLIENT_NAME}.zip"
install -d -m 700 "$OUTPUT_DIR"
[ ! -e "$BUNDLE_DIR" ] || die "安装包目录已存在"
install -d -m 700 "$BUNDLE_DIR"

cat > "$BUNDLE_DIR/${CLIENT_NAME}.conf" <<CLIENTCONF
[Interface]
PrivateKey = ${CLIENT_PRIVATE}
Address = ${CLIENT_CIDR}
MTU = 1280

[Peer]
PublicKey = ${SERVER_PUBLIC}
PresharedKey = ${PRESHARED_KEY}
Endpoint = ${PUBLIC_ENDPOINT}:${LISTEN_PORT}
AllowedIPs = ${SERVER_TUNNEL_IP}/32
PersistentKeepalive = 25
CLIENTCONF
chmod 600 "$BUNDLE_DIR/${CLIENT_NAME}.conf"

cat > "$BUNDLE_DIR/install-client.ps1" <<'POWERSHELL'
#Requires -Version 5.1
$ErrorActionPreference = 'Stop'

$TunnelName = '__TUNNEL_NAME__'
$ConfigFileName = '__CONFIG_FILE__'
$ServiceDomain = '__SERVICE_DOMAIN__'
$TunnelServerIp = '__TUNNEL_SERVER_IP__'
$WireGuardInstallerUrl = 'https://download.wireguard.com/windows-client/wireguard-installer.exe'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
$InstallRoot = Join-Path $env:ProgramData 'NewAPI-WireGuard'
$InstalledConfig = Join-Path $InstallRoot ($TunnelName + '.conf')
$HostsPath = Join-Path $env:SystemRoot 'System32\drivers\etc\hosts'
$BeginMarker = '# BEGIN NEWAPI-WIREGUARD-' + $TunnelName
$EndMarker = '# END NEWAPI-WIREGUARD-' + $TunnelName

function Write-Step([string]$Message) {
    Write-Host ('[WireGuard] ' + $Message) -ForegroundColor Cyan
}

function Test-Administrator {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

if (-not (Test-Administrator)) {
    Write-Host '正在请求管理员权限...'
    $arguments = '-NoProfile -ExecutionPolicy Bypass -File "' + $PSCommandPath + '"'
    Start-Process -FilePath 'powershell.exe' -Verb RunAs -ArgumentList $arguments
    exit 0
}

$SourceConfig = Join-Path $PSScriptRoot $ConfigFileName
if (-not (Test-Path -LiteralPath $SourceConfig)) {
    throw "缺少配置文件：$SourceConfig"
}

$WireGuardExe = Join-Path $env:ProgramFiles 'WireGuard\wireguard.exe'
if (-not (Test-Path -LiteralPath $WireGuardExe)) {
    Write-Step '下载并安装官方 WireGuard for Windows'
    $InstallerPath = Join-Path $env:TEMP 'wireguard-installer.exe'
    Invoke-WebRequest -Uri $WireGuardInstallerUrl -OutFile $InstallerPath -UseBasicParsing
    $Signature = Get-AuthenticodeSignature -FilePath $InstallerPath
    if ($Signature.Status -ne 'Valid' -or $Signature.SignerCertificate.Subject -notmatch 'O=WireGuard LLC(?:,|$)') {
        Remove-Item -LiteralPath $InstallerPath -Force -ErrorAction SilentlyContinue
        throw "WireGuard 安装包签名验证失败：$($Signature.Status)"
    }
    $Process = Start-Process -FilePath $InstallerPath -ArgumentList '/noprompt' -Wait -PassThru
    Remove-Item -LiteralPath $InstallerPath -Force -ErrorAction SilentlyContinue
    if ($Process.ExitCode -ne 0) {
        throw "WireGuard 安装失败，退出码：$($Process.ExitCode)"
    }
}

if (-not (Test-Path -LiteralPath $WireGuardExe)) {
    throw '找不到 WireGuard，请手动安装官方客户端后重新运行。'
}

Write-Step '安装分流隧道服务'
New-Item -ItemType Directory -Path $InstallRoot -Force | Out-Null
& icacls.exe $InstallRoot '/inheritance:r' '/grant:r' '*S-1-5-18:(OI)(CI)F' '*S-1-5-32-544:(OI)(CI)F' | Out-Null
if ($LASTEXITCODE -ne 0) { throw '无法保护配置目录权限' }
Copy-Item -LiteralPath $SourceConfig -Destination $InstalledConfig -Force
& icacls.exe $InstalledConfig '/inheritance:r' '/grant:r' '*S-1-5-18:F' '*S-1-5-32-544:F' | Out-Null

if ($LASTEXITCODE -ne 0) { throw '无法保护私钥文件权限' }

$ServiceName = 'WireGuardTunnel$' + $TunnelName
if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
    & $WireGuardExe /uninstalltunnelservice $TunnelName | Out-Null
    Start-Sleep -Seconds 1
}
& $WireGuardExe /installtunnelservice $InstalledConfig | Out-Null
if ($LASTEXITCODE -ne 0) { throw '安装 WireGuard 隧道服务失败' }
Start-Sleep -Seconds 3

Write-Step '写入仅此域名使用的隧道解析'
$HostsText = [IO.File]::ReadAllText($HostsPath)
Copy-Item -LiteralPath $HostsPath -Destination ($HostsPath + '.newapi-' + (Get-Date -Format 'yyyyMMddHHmmss') + '.bak')
$Pattern = '(?ms)^' + [regex]::Escape($BeginMarker) + '.*?^' + [regex]::Escape($EndMarker) + '\r?\n?'
$HostsText = [regex]::Replace($HostsText, $Pattern, '')
foreach ($Line in ($HostsText -split "`n")) {
    $Parts = (($Line -split '#', 2)[0].Trim() -split '\s+')
    if ($Parts.Length -gt 1 -and $Parts[1..($Parts.Length - 1)] -contains $ServiceDomain) {
        throw "hosts 已有 $ServiceDomain 映射，请先移除冲突条目再重试"
    }
}
if ($HostsText.Length -gt 0 -and -not $HostsText.EndsWith("`n")) {
    $HostsText += "`r`n"
}
$HostsText += $BeginMarker + "`r`n" + $TunnelServerIp + ' ' + $ServiceDomain + "`r`n" + $EndMarker + "`r`n"
[IO.File]::WriteAllText($HostsPath, $HostsText, (New-Object Text.UTF8Encoding($false)))
& ipconfig.exe /flushdns | Out-Null

Write-Step '检查隧道和 HTTPS'
$Service = Get-Service -Name $ServiceName -ErrorAction Stop
if ($Service.Status -ne 'Running') {
    Start-Service -Name $ServiceName
    Start-Sleep -Seconds 3
}

$TcpOk = Test-NetConnection -ComputerName $TunnelServerIp -Port 443 -InformationLevel Quiet
if (-not $TcpOk) {
    Write-Warning "隧道服务已安装，但 $TunnelServerIp`:443 暂不可达。请检查云平台 UDP 端口和服务端握手。"
    Write-Host '安装完成，但连通性测试未通过。' -ForegroundColor Yellow
    exit 2
}

try {
    & curl.exe --noproxy '*' --fail --silent --show-error --connect-timeout 10 --max-time 25 ('https://' + $ServiceDomain + '/api/status')
    if ($LASTEXITCODE -ne 0) { throw 'HTTPS /api/status 检查失败' }
    Write-Host ("安装成功，网站 https://{0}/，API https://{0}/v1" -f $ServiceDomain) -ForegroundColor Green
} catch {
    Write-Warning ('隧道 TCP 已连通，但 HTTPS 检查失败：' + $_.Exception.Message)
    Write-Host ('请浏览器测试：https://' + $ServiceDomain + '/') -ForegroundColor Yellow
    exit 3
}
POWERSHELL

cat > "$BUNDLE_DIR/install-client-linux.sh" <<'LINUXCLIENT'
#!/usr/bin/env bash
set -Eeuo pipefail

TUNNEL_NAME="__TUNNEL_NAME__"
CONFIG_FILE="__CONFIG_FILE__"
SERVICE_DOMAIN="__SERVICE_DOMAIN__"
TUNNEL_SERVER_IP="__TUNNEL_SERVER_IP__"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_CONFIG="$SCRIPT_DIR/$CONFIG_FILE"
TARGET_CONFIG="/etc/wireguard/${TUNNEL_NAME}.conf"
HOSTS_FILE="/etc/hosts"
BEGIN_MARKER="# BEGIN NEWAPI-WIREGUARD-${TUNNEL_NAME}"
END_MARKER="# END NEWAPI-WIREGUARD-${TUNNEL_NAME}"

log() { printf '[WireGuard] %s\n' "$*"; }
die() { printf '[WireGuard] ERROR: %s\n' "$*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "请使用 root 或 sudo 运行"
[ -f "$SOURCE_CONFIG" ] || die "缺少配置文件：$SOURCE_CONFIG"
command -v systemctl >/dev/null 2>&1 || die "系统不支持 systemd"
command -v apt-get >/dev/null 2>&1 || die "当前客户端脚本仅支持 Ubuntu/Debian"

log "安装 WireGuard"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y wireguard wireguard-tools curl ca-certificates >/dev/null

install -d -m 700 /etc/wireguard
if [ -e "$TARGET_CONFIG" ]; then
  cp -a "$TARGET_CONFIG" "${TARGET_CONFIG}.bak-$(date +%Y%m%d%H%M%S)"
fi
install -m 600 "$SOURCE_CONFIG" "$TARGET_CONFIG"

log "写入仅此域名使用的隧道解析"
awk -v begin="$BEGIN_MARKER" -v end="$END_MARKER" -v domain="$SERVICE_DOMAIN" '
  $0 == begin { own=1; next } $0 == end { own=0; next }
  !own { sub(/#.*/, ""); for (i=2; i<=NF; i++) if (tolower($i)==tolower(domain)) exit 1 }
' "$HOSTS_FILE" || die "hosts 已存在该域名映射，请先移除冲突条目再重试"
cp -a "$HOSTS_FILE" "${HOSTS_FILE}.bak-newapi-$(date +%Y%m%d%H%M%S)"
sed -i "/^${BEGIN_MARKER}$/,/^${END_MARKER}$/d" "$HOSTS_FILE"
printf '\n%s\n%s %s\n%s\n' "$BEGIN_MARKER" "$TUNNEL_SERVER_IP" "$SERVICE_DOMAIN" "$END_MARKER" >> "$HOSTS_FILE"

systemctl enable "wg-quick@${TUNNEL_NAME}" >/dev/null
systemctl restart "wg-quick@${TUNNEL_NAME}"
sleep 3

log "检查隧道和 HTTPS"
if curl --noproxy '*' -fsS --connect-timeout 10 --max-time 25 "https://${SERVICE_DOMAIN}/api/status" >/dev/null; then
  log "安装成功，网站：https://${SERVICE_DOMAIN}/  API：https://${SERVICE_DOMAIN}/v1"
else
  wg show "$TUNNEL_NAME" || true
  die "隧道已安装，但 HTTPS 检查失败；请检查服务端 UDP 端口和 WireGuard 握手"
fi
LINUXCLIENT
chmod 700 "$BUNDLE_DIR/install-client-linux.sh"

cat > "$BUNDLE_DIR/uninstall-client-linux.sh" <<'LINUXCLIENT'
#!/usr/bin/env bash
set -Eeuo pipefail
TUNNEL_NAME="__TUNNEL_NAME__"
HOSTS_FILE="/etc/hosts"
BEGIN_MARKER="# BEGIN NEWAPI-WIREGUARD-${TUNNEL_NAME}"
END_MARKER="# END NEWAPI-WIREGUARD-${TUNNEL_NAME}"
[ "$(id -u)" -eq 0 ] || { echo "请使用 root 或 sudo 运行" >&2; exit 1; }
systemctl disable --now "wg-quick@${TUNNEL_NAME}" 2>/dev/null || true
rm -f "/etc/wireguard/${TUNNEL_NAME}.conf"
sed -i "/^${BEGIN_MARKER}$/,/^${END_MARKER}$/d" "$HOSTS_FILE"
echo "WireGuard 客户隧道及域名映射已删除。"
LINUXCLIENT
chmod 700 "$BUNDLE_DIR/uninstall-client-linux.sh"

cat > "$BUNDLE_DIR/uninstall-client.ps1" <<'POWERSHELL'
#Requires -Version 5.1
$ErrorActionPreference = 'Stop'
$TunnelName = '__TUNNEL_NAME__'
$InstallRoot = Join-Path $env:ProgramData 'NewAPI-WireGuard'
$HostsPath = Join-Path $env:SystemRoot 'System32\drivers\etc\hosts'
$BeginMarker = '# BEGIN NEWAPI-WIREGUARD-' + $TunnelName
$EndMarker = '# END NEWAPI-WIREGUARD-' + $TunnelName

$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($identity)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    $arguments = '-NoProfile -ExecutionPolicy Bypass -File "' + $PSCommandPath + '"'
    Start-Process -FilePath 'powershell.exe' -Verb RunAs -ArgumentList $arguments
    exit 0
}

$WireGuardExe = Join-Path $env:ProgramFiles 'WireGuard\wireguard.exe'
if (Test-Path -LiteralPath $WireGuardExe) {
    & $WireGuardExe /uninstalltunnelservice $TunnelName | Out-Null
}

$HostsText = [IO.File]::ReadAllText($HostsPath)
Copy-Item -LiteralPath $HostsPath -Destination ($HostsPath + '.newapi-' + (Get-Date -Format 'yyyyMMddHHmmss') + '.bak')
$Pattern = '(?ms)^' + [regex]::Escape($BeginMarker) + '.*?^' + [regex]::Escape($EndMarker) + '\r?\n?'
$HostsText = [regex]::Replace($HostsText, $Pattern, '')
[IO.File]::WriteAllText($HostsPath, $HostsText, (New-Object Text.UTF8Encoding($false)))
& ipconfig.exe /flushdns | Out-Null
Remove-Item -LiteralPath (Join-Path $InstallRoot ($TunnelName + '.conf')) -Force -ErrorAction SilentlyContinue
Write-Host 'WireGuard 客户隧道及域名映射已删除。' -ForegroundColor Green
POWERSHELL

cat > "$BUNDLE_DIR/使用说明.txt" <<README
一、运行前
1. 此安装包包含该客户独立私钥，请通过可信渠道发送，不要发到群聊或公开网盘。
2. 云平台安全组需要允许 UDP ${LISTEN_PORT} 入站。
3. 测试时请关闭旧的本地 JS 代理和系统代理。
4. 一个安装包只能在一台设备上使用；多台设备需要分别生成独立密钥。

二、Windows 安装
1. 解压整个压缩包。
2. 双击 install-windows.cmd。
3. 同意管理员权限提示。
4. 脚本会安装官方 WireGuard、建立分流隧道并配置 ${DOMAIN}。
5. 成功后访问：https://${DOMAIN}/；API Base URL 为 https://${DOMAIN}/v1，继续使用原来的 API Key。
6. Python/Node SDK 在宿主机运行时同样生效。Docker、WSL 或使用代理/独立 DNS 的应用需要单独配置网络。

如果 Windows 禁止直接运行脚本，可在管理员 PowerShell 中执行：
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\install-client.ps1

三、Ubuntu/Debian 安装
在解压目录运行：
sudo bash ./install-client-linux.sh

四、卸载
Windows 管理员 PowerShell 运行：
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\uninstall-client.ps1

Ubuntu/Debian 运行：
sudo bash ./uninstall-client-linux.sh

五、安全
安装成功后请删除收到的压缩包和解压目录；不要泄露 *.conf 文件。
README

python3 - "$BUNDLE_DIR" "$CLIENT_NAME" "$DOMAIN" "$SERVER_TUNNEL_IP" <<'ENCODING'
import pathlib, sys
root, name, domain, ip = sys.argv[1:]
for path in pathlib.Path(root).iterdir():
    if path.suffix not in ('.ps1', '.sh'):
        continue
    text = path.read_text().replace('__TUNNEL_NAME__', name).replace('__CONFIG_FILE__', name + '.conf')
    text = text.replace('__SERVICE_DOMAIN__', domain).replace('__TUNNEL_SERVER_IP__', ip)
    path.write_text(text, encoding='utf-8-sig' if path.suffix == '.ps1' else 'utf-8')
ENCODING
cat > "$BUNDLE_DIR/install-windows.cmd" <<'CMD'
@echo off
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0install-client.ps1"
pause
CMD

rm -f "$ZIP_PATH"
(
  cd "$BUNDLE_DIR"
  zip -q "$ZIP_PATH" "${CLIENT_NAME}.conf" install-client.ps1 uninstall-client.ps1 install-client-linux.sh uninstall-client-linux.sh install-windows.cmd 使用说明.txt
)
chmod 600 "$ZIP_PATH"

if [ -n "${SUDO_USER:-}" ] && [ "$SUDO_USER" != "root" ]; then
  chown "$SUDO_USER":"$(id -gn "$SUDO_USER")" "$OUTPUT_DIR" "$ZIP_PATH"
  chown -R "$SUDO_USER":"$(id -gn "$SUDO_USER")" "$BUNDLE_DIR"
fi

log "服务状态：$(systemctl is-active "wg-quick@${INTERFACE}")"
log "服务端公钥：${SERVER_PUBLIC}"
log "客户端安装包：${ZIP_PATH}"
log "安装包 SHA256：$(sha256sum "$ZIP_PATH" | awk '{print $1}')"
log "请在云平台安全组放行 UDP ${LISTEN_PORT}，然后通过安全渠道把 ZIP 发给客户。"
log "客户安装后可用以下命令检查握手：sudo wg show ${INTERFACE}"

# Public bootstrap downloads contain no client keys. The provisioning code is decoded locally.
python3 - "$BUNDLE_DIR" "$CLIENT_NAME" "$DOMAIN" "$SERVER_TUNNEL_IP" "$CN_TUNNEL_VERSION" <<'ONLINE'
import base64, json, pathlib, sys
root, name, domain, ip, version = sys.argv[1:]
root = pathlib.Path(root)
payload = dict(version=version, name=name, domain=domain, ip=ip, config=(root / (name + '.conf')).read_text())
code = base64.b64encode(json.dumps(payload, separators=(',', ':')).encode()).decode()
base = 'https://github.com/nanashiwang/new-api/releases/download/cn-tunnel-v' + version
linux = '(f=$(mktemp) && curl -fLsS --connect-timeout 15 --max-time 120 ' + base + '/client-install.sh -o "$f" && sudo bash "$f" ' + code + '; r=$?; rm -f "$f"; exit "$r")'
windows = "$f=Join-Path $env:TEMP ([Guid]::NewGuid().ToString('N')+'.ps1'); try { Invoke-WebRequest '" + base + "/client-install.ps1' -UseBasicParsing -TimeoutSec 120 -OutFile $f; & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $f -Code '" + code + "'; if ($LASTEXITCODE -ne 0) { throw '安装失败，请查看上方错误' } } finally { Remove-Item -LiteralPath $f -Force -ErrorAction SilentlyContinue }"
text = '专属命令包含设备私钥，只能私下发送给该设备使用；勿公开或上传日志。\n\nWindows（管理员 PowerShell，整行复制）：\n' + windows + '\n\nUbuntu（整行复制）：\n' + linux + '\n'
(root / '一键安装命令.txt').write_text(text)
print('\n' + text)
print('命令已保存：' + str(root / '一键安装命令.txt'))
ONLINE
if [ -n "${SUDO_USER:-}" ] && [ "$SUDO_USER" != "root" ]; then
  chown "$SUDO_USER":"$(id -gn "$SUDO_USER")" "$BUNDLE_DIR/一键安装命令.txt"
fi
