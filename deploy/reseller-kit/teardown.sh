#!/usr/bin/env bash
# ============================================================================
# teardown.sh — 下线 / 退出一个代理商副站
# ----------------------------------------------------------------------------
# 用法：
#   ./teardown.sh <name> [--stop|--purge|--retarget]
#
#   --stop      （默认）停副站容器 + 主站停用批发令牌，保留数据与子域名，便于对账/恢复
#   --purge     彻底删除：停删容器与数据卷、删 Caddy 站点、主站禁用批发账户
#   --retarget  退出迁移钩子：不删数据，把子域名反代「翻向主站」，实现旧 key/旧 base_url
#               无感切换到主站（配合用户迁移搬库，详见规划文档 §6.3）
#
# 注意：--retarget 只切流量；把副站用户/余额搬进主站库是独立的迁移工程，本脚本不做。
# ============================================================================
set -euo pipefail
# 解析真实目录（支持通过 /usr/local/bin 的软链调用）
SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
# shellcheck source=lib.sh
source "$SCRIPT_DIR/lib.sh"
[ -f "$SCRIPT_DIR/reseller.env" ] || die "缺少 reseller.env"
# shellcheck source=/dev/null
set -a; source "$SCRIPT_DIR/reseller.env"; set +a
need docker jq curl

[ $# -ge 1 ] || die "用法：$0 <name> [--stop|--purge|--retarget]"
RESELLER_NAME="$1"; MODE="${2:---stop}"
validate_name "$RESELLER_NAME"
SUBSITE_CONTAINER="newapi-sub-${RESELLER_NAME}"
SUBSITE_DATA_DIR="${RESELLERS_DIR}/${RESELLER_NAME}"
WHOLESALE_USER="reseller_${RESELLER_NAME}"
load_site_env "$RESELLER_NAME"

# 主站停用/禁用批发账户，立即切断批发额度（背靠背 → 副站随之报错）
main_disable_wholesale() {
	local jar id wid resp
	jar="$(mktemp)"; id="$(api_login "$MAIN_API_URL" "$jar" "$MAIN_ADMIN_USER" "$MAIN_ADMIN_PASS")"
	wid="$(api "$MAIN_API_URL" "$jar" "$id" GET "/api/user/search?keyword=$WHOLESALE_USER" \
		| jq -r --arg n "$WHOLESALE_USER" '((.data.items // .data) // [])[]? | select(.username==$n) | .id' | head -n1)"
	if [ -n "$wid" ] && [ "$wid" != "null" ]; then
		api "$MAIN_API_URL" "$jar" "$id" POST "/api/user/manage" \
			"$(jq -nc --argjson i "$wid" '{id:$i, action:"disable"}')" >/dev/null
		log "主站批发账户已禁用 (id=$wid) → 批发额度切断"
	fi
	rm -f "$jar"
}

case "$MODE" in
  --stop)
	step "停用（保留数据）"
	docker ps --format '{{.Names}}' | grep -qx "$SUBSITE_CONTAINER" \
		&& docker compose -p "$SUBSITE_CONTAINER" -f "$SUBSITE_DATA_DIR/docker-compose.yml" stop || warn "容器未运行"
	main_disable_wholesale
	log "已停用。数据/子域名保留。恢复：重新跑 provision.sh 或 docker compose start"
	;;

  --retarget)
	step "退出迁移：子域名反代翻向主站（无感切换）"
	SUBDOMAIN="${SUBDOMAIN:?site.env 缺 SUBDOMAIN，无法 retarget}"
	# 把该子域名的反代目标改成主站端口（默认 3000），配置文件原地重渲染 + reload
	proxy_install "$SUBDOMAIN" "${MAIN_RETARGET_PORT:-3000}"
	warn "子域名已指向主站。请确认已完成「用户/余额搬库到主站 + 主站按子域名/分组计费」，否则旧 key 在主站无对应用户会鉴权失败。"
	log "随后可对该副站执行 --purge 回收资源（建议保留库快照作历史归档）"
	;;

  --purge)
	step "彻底删除"
	read -r -p "确认彻底删除副站 $RESELLER_NAME 的容器与数据？不可恢复 [yes/N] " ans
	[ "$ans" = "yes" ] || die "已取消"
	docker compose -p "$SUBSITE_CONTAINER" -f "$SUBSITE_DATA_DIR/docker-compose.yml" down -v 2>/dev/null || warn "compose down 失败/不存在"
	main_disable_wholesale
	[ -n "${SUBDOMAIN:-}" ] && proxy_remove "$SUBDOMAIN" || warn "site.env 无 SUBDOMAIN，请手动删除反代站点"
	warn "建议在删除数据目录前先备份：$SUBSITE_DATA_DIR"
	log "已删除容器/卷/反代。数据目录 $SUBSITE_DATA_DIR 保留（含 site.env），确认无需归档后再手动删除。"
	;;

  *) die "未知模式：$MODE（--stop|--purge|--retarget）" ;;
esac
