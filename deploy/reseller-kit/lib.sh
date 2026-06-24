#!/usr/bin/env bash
# ============================================================================
# reseller-kit 共享函数库（被 provision.sh / teardown.sh source）
# ============================================================================

# ---- 输出 ----
_c() { printf '\033[%sm' "$1"; }
log()  { printf '%s[ %s ]%s %s\n' "$(_c '1;32')" "$(date +%H:%M:%S)" "$(_c 0)" "$*"; }
step() { printf '\n%s>> %s%s\n' "$(_c '1;36')" "$*" "$(_c 0)"; }
warn() { printf '%s[warn]%s %s\n' "$(_c '1;33')" "$(_c 0)" "$*" >&2; }
die()  { printf '%s[fail]%s %s\n' "$(_c '1;31')" "$(_c 0)" "$*" >&2; exit 1; }

# ---- 依赖检查 ----
need() { for c in "$@"; do command -v "$c" >/dev/null 2>&1 || die "缺少依赖命令：$c"; done; }

# ---- 随机串 / 额度换算 ----
rand_secret() { openssl rand -hex 32; }
# 美元 → quota（1 美元 = QuotaPerUnit，默认 500000，见 common/constants.go:25）
usd_to_quota() { awk -v u="$1" 'BEGIN{ printf "%d", u*500000 }'; }

# ---- 名称 / 子域名校验 ----
validate_name() {
	[[ "$1" =~ ^[a-z0-9][a-z0-9_-]{1,30}$ ]] || die "代理商标识非法（只允许小写字母/数字/_-，2-31 位）：$1"
}
validate_subdomain() {
	[[ "$1" =~ ^[a-z0-9.-]+\.[a-z]{2,}$ ]] || die "子域名非法：$1"
}

# ---- 端口分配（仅查 127.0.0.1） ----
_port_in_use() { (exec 3<>"/dev/tcp/127.0.0.1/$1") >/dev/null 2>&1 && { exec 3>&- 3<&- 2>/dev/null; return 0; }; return 1; }
free_port() {
	local p
	for p in $(seq "${PORT_RANGE_START:-3101}" "${PORT_RANGE_END:-3999}"); do
		_port_in_use "$p" || { echo "$p"; return 0; }
	done
	die "端口区间 ${PORT_RANGE_START}-${PORT_RANGE_END} 内无可用端口"
}

# ============================================================================
# new-api 管理 API 封装
# 鉴权（已核实 middleware/auth.go:33 authHelper）：
#   - 登录后用 cookie jar；每次请求带 New-API-User: <user_id> 头（强制）。
# ============================================================================

# login <base> <cookiejar> <user> <pass>  → 回显登录用户 id
api_login() {
	local base=$1 jar=$2 user=$3 pass=$4 resp
	resp=$(curl -sS -c "$jar" -X POST "$base/api/user/login" \
		-H 'Content-Type: application/json' \
		-d "$(jq -nc --arg u "$user" --arg p "$pass" '{username:$u,password:$p}')") \
		|| die "登录请求失败：$base"
	[ "$(jq -r '.success' <<<"$resp")" = "true" ] \
		|| die "登录失败（$base，用户 $user）：$(jq -r '.message' <<<"$resp")"
	jq -r '.data.id' <<<"$resp"
}

# api <base> <jar> <uid> <method> <path> [json_body]  → 回显响应体；失败即 die
api() {
	local base=$1 jar=$2 uid=$3 method=$4 path=$5 body=${6:-} resp
	local args=(-sS -X "$method" "$base$path" -H "New-API-User: $uid" -b "$jar")
	[ -n "$body" ] && args+=(-H 'Content-Type: application/json' -d "$body")
	resp=$(curl "${args[@]}") || die "请求失败：$method $path"
	if [ "$(jq -r '.success // empty' <<<"$resp")" = "false" ]; then
		die "API 返回失败：$method $path → $(jq -r '.message' <<<"$resp")"
	fi
	printf '%s' "$resp"
}

# api_tok <base> <access_token> <uid> <method> <path> [json]  → 用访问令牌调管理 API
# 优势：① 绕过登录的 Turnstile 人机验证；② access_token 属于 root 账号即拥有 root 权限(/api/option)。
api_tok() {
	local base=$1 tok=$2 uid=$3 method=$4 path=$5 body=${6:-} resp
	local args=(-sS -X "$method" "$base$path" -H "Authorization: Bearer $tok" -H "New-API-User: $uid")
	[ -n "$body" ] && args+=(-H 'Content-Type: application/json' -d "$body")
	resp=$(curl "${args[@]}") || die "请求失败：$method $path"
	if [ "$(jq -r '.success // empty' <<<"$resp")" = "false" ]; then
		die "API 返回失败：$method $path → $(jq -r '.message' <<<"$resp")"
	fi
	printf '%s' "$resp"
}

# 等待站点健康（GET /api/status 返回 success:true，见 controller/misc.go）
wait_health() {
	local url=$1 i
	for i in $(seq 1 60); do
		if curl -sf "$url/api/status" 2>/dev/null | grep -q '"success":\s*true'; then
			log "站点就绪：$url"; return 0
		fi
		sleep 2
	done
	die "等待站点就绪超时：$url"
}

# 读/写 每副站的 site.env（保存端口、密钥、密码、批发 token 等，供再次运行与运维查阅）
site_env_path() { echo "${RESELLERS_DIR}/$1/site.env"; }
load_site_env()  { local f; f=$(site_env_path "$1"); [ -f "$f" ] && set -a && . "$f" && set +a || true; }
save_site_kv()   { # <name> <key> <value>
	local f; f=$(site_env_path "$1"); mkdir -p "$(dirname "$f")"; touch "$f"; chmod 600 "$f"
	grep -v "^$2=" "$f" > "$f.tmp" 2>/dev/null || true; mv "$f.tmp" "$f"
	printf '%s=%q\n' "$2" "$3" >> "$f"
}

# ============================================================================
# 反向代理（nginx / caddy，由 reseller.env 的 PROXY_KIND 决定；none = 跳过自己配）
# 依赖调用方已定义 SCRIPT_DIR（模板所在目录）。
# ============================================================================
proxy_site_file() { # <subdomain> → 站点配置文件路径
	case "${PROXY_KIND:-nginx}" in
		nginx) echo "${NGINX_SITES_DIR}/$1.conf" ;;
		caddy) echo "${CADDY_SITES_DIR}/$1.caddy" ;;
		none)  echo "" ;;
		*) die "未知 PROXY_KIND：${PROXY_KIND}" ;;
	esac
}
proxy_reload() {
	case "${PROXY_KIND:-nginx}" in
		nginx) eval "${NGINX_RELOAD_CMD:-nginx -t && systemctl reload nginx}" ;;
		caddy) eval "${CADDY_RELOAD_CMD:-systemctl reload caddy}" ;;
		none)  : ;;
	esac
}
# proxy_install <subdomain> <port>：渲染站点配置并 reload
proxy_install() {
	local sub=$1 port=$2 tpl dir out
	case "${PROXY_KIND:-nginx}" in
		nginx) dir="$NGINX_SITES_DIR"; tpl="$SCRIPT_DIR/nginx-site.conf.template" ;;
		caddy) dir="$CADDY_SITES_DIR"; tpl="$SCRIPT_DIR/Caddyfile.snippet.template" ;;
		none)  warn "PROXY_KIND=none：请手动把 $sub 反代到 127.0.0.1:$port"; return 0 ;;
	esac
	[ -d "$dir" ] || { warn "代理目录不存在（$dir），跳过——请手动把 $sub 反代到 127.0.0.1:$port"; return 0; }
	out="$(proxy_site_file "$sub")"
	export SUBDOMAIN="$sub" SUBSITE_PORT="$port" TLS_CERT="${TLS_CERT:-}" TLS_KEY="${TLS_KEY:-}"
	# 受限 envsubst：只替换我们的占位符，保留 nginx 的 $host / $remote_addr 等
	envsubst '${SUBDOMAIN} ${SUBSITE_PORT} ${TLS_CERT} ${TLS_KEY}' < "$tpl" > "$out"
	if proxy_reload; then log "反代已加载：$sub → 127.0.0.1:$port（$PROXY_KIND）"
	else warn "反代 reload 失败，请手动检查：$out"; fi
}
# proxy_remove <subdomain>：删除站点并 reload
proxy_remove() {
	local f; f="$(proxy_site_file "$1")"
	[ -n "$f" ] && [ -f "$f" ] && { rm -f "$f"; proxy_reload; log "已删除反代站点：$1"; } || true
}
