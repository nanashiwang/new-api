#!/usr/bin/env bash
# ============================================================================
# provision.sh — 一键开通一个代理商副站（方案 C：副站独立实例 + 上游回指主站）
# ----------------------------------------------------------------------------
# 用法：
#   ./provision.sh <name> <subdomain> [prepaid_usd]
# 例：
#   ./provision.sh acme acme.example.com 20
#
# 计费模式：1:1 复刻(auto+原价)——代理商按你的原价用你现有的分组，不另设折扣分组、不改渠道。
# 做的事（宿主层 + app 层，全部基于已核实的真实 API 契约）：
#   1. 起副站容器（独立库/独立密钥，仅绑 127.0.0.1）
#   2. 配子域名反代（nginx/caddy）
#   3. 主站：确保启用 auto 分组 → 建批发账户(放标准客户组) → 设额度 → 建 auto 令牌
#   4. 副站：建「回指主站」渠道（key=批发令牌）→ 改默认 root 密码
# ============================================================================
set -euo pipefail
# 解析真实目录（支持通过 /usr/local/bin 的软链调用）
SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
# shellcheck source=lib.sh
source "$SCRIPT_DIR/lib.sh"

[ -f "$SCRIPT_DIR/reseller.env" ] || die "缺少 reseller.env（请复制 reseller.env.example 并填写）"
# shellcheck source=/dev/null
set -a; source "$SCRIPT_DIR/reseller.env"; set +a

need docker jq curl envsubst openssl awk

# ---------- 参数 ----------
[ $# -ge 2 ] || die "用法：$0 <name> <subdomain> [预付美元]"
RESELLER_NAME="$1"; SUBDOMAIN="$2"
PREPAID_USD="${3:-$DEFAULT_PREPAID_USD}"
validate_name "$RESELLER_NAME"; validate_subdomain "$SUBDOMAIN"

RESELLER_BILLING_GROUP="${RESELLER_BILLING_GROUP:-default}"   # 批发账号计费分组=你的标准客户组 → 代理商付同价
SUBSITE_CONTAINER="newapi-sub-${RESELLER_NAME}"
SUBSITE_DATA_DIR="${RESELLERS_DIR}/${RESELLER_NAME}"
WHOLESALE_USER="reseller_${RESELLER_NAME}"
QUOTA="$(usd_to_quota "$PREPAID_USD")"

log "开通代理商：name=$RESELLER_NAME  子域名=$SUBDOMAIN  计费分组=$RESELLER_BILLING_GROUP(原价)  预付=\$$PREPAID_USD ($QUOTA quota)"
load_site_env "$RESELLER_NAME"   # 再次运行时复用已有密钥/密码（幂等）

# ============================================================================
step "1/6 起副站容器"
# ============================================================================
if docker ps -a --format '{{.Names}}' | grep -qx "$SUBSITE_CONTAINER"; then
	warn "容器 $SUBSITE_CONTAINER 已存在，跳过创建"
	SUBSITE_PORT="${SUBSITE_PORT:?site.env 缺少 SUBSITE_PORT，请手动确认端口}"
else
	SUBSITE_PORT="${SUBSITE_PORT:-$(free_port)}"
	SESSION_SECRET="${SESSION_SECRET:-$(rand_secret)}"
	CRYPTO_SECRET="${CRYPTO_SECRET:-$(rand_secret)}"
	mkdir -p "$SUBSITE_DATA_DIR/data" "$SUBSITE_DATA_DIR/logs"
	export RESELLER_NAME SUBSITE_CONTAINER SUBSITE_PORT SUBSITE_DATA_DIR \
	       SESSION_SECRET CRYPTO_SECRET TZ NEWAPI_IMAGE SHARED_NETWORK
	envsubst < "$SCRIPT_DIR/compose.template.yml" > "$SUBSITE_DATA_DIR/docker-compose.yml"
	save_site_kv "$RESELLER_NAME" SUBSITE_PORT "$SUBSITE_PORT"
	save_site_kv "$RESELLER_NAME" SESSION_SECRET "$SESSION_SECRET"
	save_site_kv "$RESELLER_NAME" CRYPTO_SECRET "$CRYPTO_SECRET"
	docker compose -p "$SUBSITE_CONTAINER" -f "$SUBSITE_DATA_DIR/docker-compose.yml" up -d
	log "容器已启动，端口 127.0.0.1:$SUBSITE_PORT"
fi
SUBSITE_API="http://127.0.0.1:${SUBSITE_PORT}"
wait_health "$SUBSITE_API"

# ============================================================================
step "2/6 配置子域名反代 + 证书"
# ============================================================================
proxy_install "$SUBDOMAIN" "$SUBSITE_PORT"
save_site_kv "$RESELLER_NAME" SUBDOMAIN "$SUBDOMAIN"

# ============================================================================
step "3/6 主站：登录 + 启用 auto 分组(1:1 原价模式)"
# ============================================================================
MAIN_JAR="$(mktemp)"; trap 'rm -f "$MAIN_JAR" "${WS_JAR:-}" "${SUB_JAR:-}"' EXIT
MAIN_ID="$(api_login "$MAIN_API_URL" "$MAIN_JAR" "$MAIN_ADMIN_USER" "$MAIN_ADMIN_PASS")"
log "主站管理员登录成功 (id=$MAIN_ID)"

OPTS="$(api "$MAIN_API_URL" "$MAIN_JAR" "$MAIN_ID" GET "/api/option/")"
# (a) UserUsableGroups 确保含 "auto"——否则 auto 令牌会被拒(auth.go:345)。value 以字符串提交。
CUR_UUG="$(jq -r '(.data[]? | select(.key=="UserUsableGroups") | .value) // "{}"' <<<"$OPTS")"
NEW_UUG="$(jq -c '. + {"auto":"自动(按模型选分组, 1:1原价)"}' <<<"$CUR_UUG")"
api "$MAIN_API_URL" "$MAIN_JAR" "$MAIN_ID" PUT "/api/option/" \
	"$(jq -nc --arg v "$NEW_UUG" '{key:"UserUsableGroups", value:$v}')" >/dev/null
# (b) AutoGroups 并入要开放给代理商的分组(逗号→数组, 去重)——auto 只在这些分组里按模型选。
CUR_AG="$(jq -r '(.data[]? | select(.key=="AutoGroups") | .value) // "[\"default\"]"' <<<"$OPTS")"
NEW_AG="$(jq -c --arg g "$AUTO_GROUPS" '(. + ($g | split(",") | map(gsub("^ +| +$";"")))) | unique' <<<"$CUR_AG")"
api "$MAIN_API_URL" "$MAIN_JAR" "$MAIN_ID" PUT "/api/option/" \
	"$(jq -nc --arg v "$NEW_AG" '{key:"AutoGroups", value:$v}')" >/dev/null
log "auto 已启用；开放分组并入 AutoGroups：$AUTO_GROUPS"

# ============================================================================
step "4/6 主站：建批发账户(标准客户组) + 设额度 + 建 auto 令牌"
# ============================================================================
get_uid() { api "$MAIN_API_URL" "$MAIN_JAR" "$MAIN_ID" GET "/api/user/search?keyword=$1" \
	| jq -r --arg n "$1" '((.data.items // .data) // [])[]? | select(.username==$n) | .id' | head -n1; }

WHOLESALE_UID="$(get_uid "$WHOLESALE_USER" || true)"
if [ -z "${WHOLESALE_UID:-}" ] || [ "$WHOLESALE_UID" = "null" ]; then
	WHOLESALE_PASS="${WHOLESALE_PASS:-$(rand_secret | cut -c1-20)}"
	save_site_kv "$RESELLER_NAME" WHOLESALE_PASS "$WHOLESALE_PASS"
	# CreateUser 不支持直接设 role/quota/group（见契约），先建再 PUT
	api "$MAIN_API_URL" "$MAIN_JAR" "$MAIN_ID" POST "/api/user/" \
		"$(jq -nc --arg u "$WHOLESALE_USER" --arg p "$WHOLESALE_PASS" --arg d "Reseller $RESELLER_NAME 批发账户" \
			'{username:$u, password:$p, display_name:$d}')" >/dev/null
	WHOLESALE_UID="$(get_uid "$WHOLESALE_USER")"
	log "已建批发账户 $WHOLESALE_USER (id=$WHOLESALE_UID)"
else
	WHOLESALE_PASS="${WHOLESALE_PASS:?批发账户已存在但 site.env 缺 WHOLESALE_PASS，无法登录建令牌}"
	log "批发账户已存在 (id=$WHOLESALE_UID)，复用"
fi
save_site_kv "$RESELLER_NAME" WHOLESALE_UID "$WHOLESALE_UID"

# PUT 更新：放标准客户组 + 预付额度 + role=1（必须显式带 role，否则默认 0 降为游客）
api "$MAIN_API_URL" "$MAIN_JAR" "$MAIN_ID" PUT "/api/user/" \
	"$(jq -nc --argjson id "$WHOLESALE_UID" --arg u "$WHOLESALE_USER" --arg d "Reseller $RESELLER_NAME 批发账户" \
		--arg g "$RESELLER_BILLING_GROUP" --argjson q "$QUOTA" \
		'{id:$id, username:$u, display_name:$d, group:$g, quota:$q, role:1}')" >/dev/null
log "批发账户已设：group=$RESELLER_BILLING_GROUP(原价)  quota=$QUOTA"

# 令牌创建需「以批发账户本人」登录（POST /api/token/ 走 UserAuth）
WS_JAR="$(mktemp)"
WS_ID="$(api_login "$MAIN_API_URL" "$WS_JAR" "$WHOLESALE_USER" "$WHOLESALE_PASS")"
TOKEN_NAME="reseller-${RESELLER_NAME}-wholesale"
# 已有同名令牌则复用其 key（幂等）
TOKEN_ID="$(api "$MAIN_API_URL" "$WS_JAR" "$WS_ID" GET "/api/token/?p=1&page_size=100" \
	| jq -r --arg n "$TOKEN_NAME" '((.data.items // .data) // [])[]? | select(.name==$n) | .id' | head -n1 || true)"
if [ -z "${TOKEN_ID:-}" ] || [ "$TOKEN_ID" = "null" ]; then
	# group="auto"：令牌按请求模型在 AUTO_GROUPS 内自动选分组、按各自原价计费(1:1)。
	# MODELS 为空 → 不限制模型(全部)；非空 → 限定到 MODELS。
	if [ -n "${MODELS:-}" ]; then TOK_LE=true; TOK_LM="$MODELS"; else TOK_LE=false; TOK_LM=""; fi
	api "$MAIN_API_URL" "$WS_JAR" "$WS_ID" POST "/api/token/" \
		"$(jq -nc --arg n "$TOKEN_NAME" --argjson le "$TOK_LE" --arg m "$TOK_LM" \
			'{name:$n, unlimited_quota:true, remain_quota:0, expired_time:-1,
			  model_limits_enabled:$le, model_limits:$m,
			  channel_limits_enabled:false, channel_limits:"",
			  allow_ips:"", group:"auto", cross_group_retry:true,
			  max_concurrency:0, window_request_limit:0, window_seconds:0,
			  package_enabled:false, package_limit_quota:0, package_period:"none",
			  package_period_mode:"relative", package_custom_seconds:0,
			  package_used_quota:0, package_next_reset_time:0}')" >/dev/null
	TOKEN_ID="$(api "$MAIN_API_URL" "$WS_JAR" "$WS_ID" GET "/api/token/?p=1&page_size=100" \
		| jq -r --arg n "$TOKEN_NAME" '((.data.items // .data) // [])[]? | select(.name==$n) | .id' | head -n1)"
	log "已建批发令牌 (id=$TOKEN_ID, unlimited, 限定模型)"
fi
# 取完整 key（GET /api/token/{id}/key 仅本人可调）
WTOKEN_KEY="$(api "$MAIN_API_URL" "$WS_JAR" "$WS_ID" GET "/api/token/${TOKEN_ID}/key" | jq -r '.data // .key')"
[ -n "$WTOKEN_KEY" ] && [ "$WTOKEN_KEY" != "null" ] || die "未取到批发令牌 key"
save_site_kv "$RESELLER_NAME" WTOKEN_KEY "$WTOKEN_KEY"

# 副站回指渠道的模型列表：MODELS 指定则用之；为空则用批发令牌从主站 /v1/models 拉全量
if [ -n "${MODELS:-}" ]; then
	CHANNEL_MODELS="$MODELS"
else
	log "MODELS 为空 → 从主站 /v1/models 拉取该令牌可用的全部模型..."
	CHANNEL_MODELS="$(curl -sS -H "Authorization: Bearer $WTOKEN_KEY" "$MAIN_API_URL/v1/models" | jq -r '[.data[]?.id] | join(",")')"
	[ -n "$CHANNEL_MODELS" ] || die "拉取主站模型列表失败(确认 auto/AUTO_GROUPS 已生效)"
	log "拉到 $(printf '%s' "$CHANNEL_MODELS" | tr ',' '\n' | grep -c .) 个模型"
fi

# ============================================================================
step "5/6 副站：建「回指主站」渠道"
# ============================================================================
SUB_ROOT_PASS_CUR="${SUB_ROOT_PASS:-123456}"   # 首启默认 root/123456；改密后从 site.env 读
SUB_JAR="$(mktemp)"
SUB_ID="$(api_login "$SUBSITE_API" "$SUB_JAR" "root" "$SUB_ROOT_PASS_CUR")"
CH_NAME="上游-主站(wholesale)"
EXIST_CH="$(api "$SUBSITE_API" "$SUB_JAR" "$SUB_ID" GET "/api/channel/?p=1&page_size=100" \
	| jq -r --arg n "$CH_NAME" '((.data.items // .data) // [])[]? | select(.name==$n) | .id' | head -n1 || true)"
if [ -z "${EXIST_CH:-}" ] || [ "$EXIST_CH" = "null" ]; then
	# type=1 (ChannelTypeOpenAI)；base_url=主站内网；models 逗号串；建渠道自动建 ability
	api "$SUBSITE_API" "$SUB_JAR" "$SUB_ID" POST "/api/channel/" \
		"$(jq -nc --arg n "$CH_NAME" --arg k "$WTOKEN_KEY" --arg b "$MAIN_INTERNAL_URL" --arg m "$CHANNEL_MODELS" \
			'{mode:"single", channel:{name:$n, type:1, key:$k, base_url:$b, models:$m,
			  group:"default", priority:100, weight:0, status:1}}')" >/dev/null
	log "副站回指渠道已建：base_url=$MAIN_INTERNAL_URL"
else
	warn "副站已存在渠道「$CH_NAME」(id=$EXIST_CH)，跳过"
fi

# ============================================================================
step "6/6 副站：修改默认 root 密码"
# ============================================================================
if [ "$SUB_ROOT_PASS_CUR" = "123456" ]; then
	NEW_ROOT_PASS="$(rand_secret | cut -c1-20)"
	api "$SUBSITE_API" "$SUB_JAR" "$SUB_ID" PUT "/api/user/self" \
		"$(jq -nc --arg o "123456" --arg p "$NEW_ROOT_PASS" '{original_password:$o, password:$p}')" >/dev/null
	save_site_kv "$RESELLER_NAME" SUB_ROOT_PASS "$NEW_ROOT_PASS"
	log "副站 root 默认密码已修改"
else
	NEW_ROOT_PASS="$SUB_ROOT_PASS_CUR"; log "副站 root 密码此前已修改，跳过"
fi

# ============================================================================
cat <<EOF

$(_c '1;32')========== 开通完成 ==========$(_c 0)
代理商：        $RESELLER_NAME
副站地址：      https://$SUBDOMAIN   （本地: $SUBSITE_API）
副站 root 密码：$NEW_ROOT_PASS    ← 交付给代理商，登录后请再次自行修改
批发账户：      $WHOLESALE_USER  (主站 id=$WHOLESALE_UID, 计费分组 $RESELLER_BILLING_GROUP=原价)
开放分组(auto)：$AUTO_GROUPS
预付批发额度：  \$$PREPAID_USD ($QUOTA quota)
机密档案：      $(site_env_path "$RESELLER_NAME")  （含密钥/密码/批发token，chmod 600，妥善保管）

下一步验证（跑通双向扣费）：
  1. 在副站建一个普通用户+令牌，调一次模型（副站渠道已含 $(printf '%s' "$CHANNEL_MODELS" | tr ',' '\n' | grep -c .) 个模型）；
  2. 副站「日志」看到一笔零售消费；
  3. 主站「日志」按 user_id=$WHOLESALE_UID 看到一笔批发消费 → 闭环成立。
EOF
