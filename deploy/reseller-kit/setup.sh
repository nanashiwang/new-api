#!/usr/bin/env bash
# ============================================================================
# setup.sh — 一次性安装（装依赖 + 向导生成 reseller.env + 注册全局命令）
# ----------------------------------------------------------------------------
# 用法：  sudo bash setup.sh
# 装完后，开通副站只要一行：
#         sudo newapi-reseller <name> <subdomain> [批发倍率] [预付美元]
# ============================================================================
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
# shellcheck source=lib.sh
source "$SCRIPT_DIR/lib.sh"

[ "$(id -u)" -eq 0 ] || die "请用 root 运行：  sudo bash setup.sh"

step "1/4 安装依赖"
if command -v apt-get >/dev/null 2>&1; then
	apt-get update -qq
	apt-get install -y -qq jq curl gettext-base openssl
	log "依赖就绪（jq curl envsubst openssl）"
else
	need jq curl envsubst openssl   # 非 apt 系统：请自行确保已装上述命令
fi
command -v docker >/dev/null 2>&1 || die "未找到 docker（主站就是用它跑的，请先装好）"

step "2/4 生成 reseller.env（配置向导）"
ENV_FILE="$SCRIPT_DIR/reseller.env"
if [ -f "$ENV_FILE" ]; then
	log "reseller.env 已存在，跳过向导（如需重配：删除它后重跑 setup.sh）"
else
	echo "（直接回车用 [] 内默认值）"
	# 自动探测主站容器（名字含 new-api/newapi、排除副站 sub-）
	DETECTED_MAIN="$(docker ps --format '{{.Names}}' | grep -iE 'new-?api' | grep -v 'sub-' | head -n1 || true)"
	read -r -p "主站容器名 [${DETECTED_MAIN:-new-api}]: " MAIN_CONTAINER
	MAIN_CONTAINER="${MAIN_CONTAINER:-${DETECTED_MAIN:-new-api}}"
	DETECTED_NET="$(docker inspect "$MAIN_CONTAINER" \
		--format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{end}}' 2>/dev/null | head -n1 || true)"
	DETECTED_IMG="$(docker inspect "$MAIN_CONTAINER" --format '{{.Config.Image}}' 2>/dev/null || true)"
	read -r -p "主站 docker 网络名 [${DETECTED_NET:-newapi_default}]: " IN_NET
	IN_NET="${IN_NET:-${DETECTED_NET:-newapi_default}}"
	read -r -p "副站镜像（建议与主站一致）[${DETECTED_IMG:-calciumion/new-api:latest}]: " IN_IMG
	IN_IMG="${IN_IMG:-${DETECTED_IMG:-calciumion/new-api:latest}}"
	read -r -p "主站管理API地址(脚本从宿主访问) [http://127.0.0.1:3000]: " IN_APIURL
	IN_APIURL="${IN_APIURL:-http://127.0.0.1:3000}"
	read -r -p "你的根域名（如 example.com）: " IN_DOMAIN
	[ -n "$IN_DOMAIN" ] || die "域名不能为空"
	read -r -p "主站管理员账号（role>=10，勿用 root）: " IN_ADMIN
	read -r -s -p "主站管理员密码: " IN_PASS; echo
	[ -n "$IN_ADMIN" ] && [ -n "$IN_PASS" ] || die "管理员账号/密码不能为空"
	read -r -p "开放给代理商转售的模型（逗号分隔，留空=全部模型）: " IN_MODELS
	read -r -p "开放给代理商的分组(逗号分隔,填你现有的分组) [vip,svip,claude,gemini,kiro]: " IN_GROUPS
	IN_GROUPS="${IN_GROUPS:-vip,svip,claude,gemini,kiro}"
	read -r -p "默认预付额度(美元) [10]: " IN_USD; IN_USD="${IN_USD:-10}"
	read -r -p "nginx 站点目录 [/etc/nginx/conf.d]: " IN_NGX; IN_NGX="${IN_NGX:-/etc/nginx/conf.d}"

	# 用 printf %q 写入，安全转义密码等特殊字符
	{
		echo "# reseller.env — 由 setup.sh 生成；chmod 600；切勿入库"
		printf 'MAIN_API_URL=%q\n'           "$IN_APIURL"
		printf 'MAIN_INTERNAL_URL=%q\n'      "http://$MAIN_CONTAINER:3000"
		printf 'MAIN_ADMIN_USER=%q\n'        "$IN_ADMIN"
		printf 'MAIN_ADMIN_PASS=%q\n'        "$IN_PASS"
		printf 'SHARED_NETWORK=%q\n'         "$IN_NET"
		printf 'NEWAPI_IMAGE=%q\n'           "$IN_IMG"
		printf 'RESELLERS_DIR=%q\n'          "/opt/newapi-resellers"
		printf 'PORT_RANGE_START=%q\n'       "3101"
		printf 'PORT_RANGE_END=%q\n'         "3999"
		printf 'TZ=%q\n'                     "Asia/Shanghai"
		printf 'PROXY_KIND=%q\n'             "nginx"
		printf 'NGINX_SITES_DIR=%q\n'        "$IN_NGX"
		printf 'NGINX_RELOAD_CMD=%q\n'       "nginx -t && systemctl reload nginx"
		printf 'TLS_CERT=%q\n'               "/etc/letsencrypt/live/$IN_DOMAIN/fullchain.pem"
		printf 'TLS_KEY=%q\n'                "/etc/letsencrypt/live/$IN_DOMAIN/privkey.pem"
		printf 'RESELLER_BILLING_GROUP=%q\n' "default"
		printf 'AUTO_GROUPS=%q\n'            "$IN_GROUPS"
		printf 'DEFAULT_PREPAID_USD=%q\n'    "$IN_USD"
		printf 'MODELS=%q\n'                 "$IN_MODELS"
	} > "$ENV_FILE"
	chmod 600 "$ENV_FILE"
	export SETUP_DOMAIN="$IN_DOMAIN"
	log "已生成 $ENV_FILE"
fi

step "3/4 注册全局命令"
chmod +x "$SCRIPT_DIR/provision.sh" "$SCRIPT_DIR/teardown.sh" "$SCRIPT_DIR/lib.sh"
ln -sf "$SCRIPT_DIR/provision.sh" /usr/local/bin/newapi-reseller
ln -sf "$SCRIPT_DIR/teardown.sh"  /usr/local/bin/newapi-reseller-rm
log "已注册命令： newapi-reseller / newapi-reseller-rm"

step "4/4 完成"
DOMAIN_HINT="${SETUP_DOMAIN:-你的域名}"
cat <<EOF

$(_c '1;32')========== 安装完成 ==========$(_c 0)
计费模式：1:1 复刻(原价)——代理商按你的原价用你现有分组，无需建折扣分组、无需改渠道；
首次开通时脚本会自动启用 auto 分组、并把开放分组并入 AutoGroups。

还差一件「一次性」事：申请泛域名证书
   sudo certbot certonly --dns-<你的DNS商> -d "*.${DOMAIN_HINT}" -d "${DOMAIN_HINT}"

然后，开通一个代理商就一行：

   $(_c '1;36')sudo newapi-reseller <name> <subdomain> [预付美元]$(_c 0)
   例：sudo newapi-reseller acme acme.${DOMAIN_HINT} 20

下线/退出：
   sudo newapi-reseller-rm <name> --stop      # 停用(保留数据,可恢复)
   sudo newapi-reseller-rm <name> --purge     # 彻底删除
   sudo newapi-reseller-rm <name> --retarget  # 子域名翻向主站(无感迁移)
EOF
