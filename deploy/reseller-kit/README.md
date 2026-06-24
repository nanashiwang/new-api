# reseller-kit —— 代理商副站「一键开通」脚本包

> 配套规划：`docs/reseller-distribution-plan.md`（方案 C）
> 形态：运维在**主站同一台服务器**上「复制即开通」一个代理商副站。零改 new-api 核心代码、不挂 docker.sock。

---

## 🚀 三步上手

```bash
# ① 一次性安装（装依赖 + 配置向导 + 注册全局命令）
cd <你的 newapi 仓库>/deploy/reseller-kit
sudo bash setup.sh

# ② 一次性主站设置（只做一次，管所有代理商）
#    a. 主站后台 → 渠道 → 给要开放的上游渠道「分组」加上 reseller_wholesale → 保存
#    b. 申请泛域名证书： sudo certbot certonly --dns-<你的DNS商> -d "*.你的域名" -d "你的域名"

# ③ 开通一个代理商 —— 就这一行 👇
sudo newapi-reseller acme acme.你的域名 0.7 20
```

参数：`newapi-reseller <name> <subdomain> [批发倍率] [预付美元]`，后两个可省略（用向导里设的默认值）。

---

## 它做什么

一行命令开通一个代理商：起独立副站容器（独立库/独立密钥）→ 配 nginx 子域名 → 在主站建好「批发账户 + 批发分组折扣 + unlimited 批发令牌」→ 在副站建好「回指主站」渠道 → 改默认密码。

计费模型（三层价，全部复用现有 `model_ratio × group_ratio`）：

```
下游用户 ─付费(零售价,副站定)→ 副站 ─批发令牌→ 主站 ─批发价=model_ratio×批发折扣→ 各原厂
                                         主站从「批发账户钱包」扣额度，耗尽即背靠背断流
```

## 目录文件

| 文件 | 作用 |
|---|---|
| `setup.sh` | 一次性安装：装依赖、配置向导、注册全局命令 `newapi-reseller` |
| `provision.sh` | 开通逻辑（被全局命令 `newapi-reseller` 调用） |
| `teardown.sh` | 下线逻辑（被 `newapi-reseller-rm` 调用） |
| `lib.sh` | 共享函数（API 封装、反代抽象、端口分配、site.env 读写） |
| `reseller.env.example` | 配置模板（`setup.sh` 会据此向导生成 `reseller.env`） |
| `nginx-site.conf.template` / `Caddyfile.snippet.template` | 反代站点模板（nginx 默认 / Caddy 备选） |
| `compose.template.yml` | 副站容器模板 |

## 前置条件

1. **主站**已用 docker compose 跑起来（`container_name: new-api`，宿主已映射 `3000`）。
2. 一个**主站专用管理员账号**（role≥10，别用 root）。
3. 服务器有 nginx（或 Caddy）做反代；有 docker（compose v2）。
4. `setup.sh` 会自动装 `jq curl gettext-base openssl`（Debian/Ubuntu）。

### ⚠️ 一次性主站设置（不做会「无可用渠道」）

批发请求用「批发分组」去主站**选渠道**，所以该分组**必须挂到上游渠道**上：

- 主站后台 → **渠道** → 编辑要开放给代理商的每个上游渠道 → 「**分组**」里**追加** `reseller_wholesale`（保留原有 default 等）→ 保存。
- 分组的折扣倍率（`group_ratio`）由脚本首次开通时自动写入，无需手动。

> 只给部分渠道加 → 代理商就只能转售这些渠道的模型（配合批发令牌的 `model_limits` 双保险）。

## 验证「双向扣费」闭环（开通后必做）

1. 浏览器开 `https://<subdomain>`，用打印的 root 密码登录 → 建一个普通用户 + 令牌。
2. 用该令牌调一次模型（base_url 指向子域名）。
3. **副站**「日志」出现一笔**零售**消费；**主站**「日志」按批发账户 id 出现一笔**批发**消费 → 闭环成立。

## 给代理商续费批发额度

代理商付款给你后，到主站后台把「批发账户」额度调高（或走主站充值）。额度 = 美元 × 500000。

## 下线 / 退出

```bash
sudo newapi-reseller-rm <name> --stop       # 停容器+停用批发令牌，保留数据(默认,可恢复)
sudo newapi-reseller-rm <name> --purge      # 彻底删除容器/卷/反代(交互确认)
sudo newapi-reseller-rm <name> --retarget   # 退出迁移：子域名反代翻向主站(无感切换钩子)
```

> `--retarget` 只切流量。要让旧 key 在主站继续可用，需先把副站 `users`/`tokens`/余额**搬进主站库**——独立迁移工程（规划文档 §6，阶段 2），本 kit 不含。

## 手动配置（不走向导时）

`cp reseller.env.example reseller.env && chmod 600 reseller.env`，按注释填好后 `sudo ./provision.sh <name> <subdomain>`。
切换 nginx/Caddy：改 `reseller.env` 的 `PROXY_KIND`（`nginx`|`caddy`|`none`）。

## 设计要点（与真实代码对应）

| 环节 | 真实依据 |
|---|---|
| 鉴权 | 登录拿 cookie + 每请求带 `New-API-User` 头（`middleware/auth.go:33`） |
| 批发计费分组 | 令牌 `group=批发分组`（与账户 group 一致）；用户自身分组必在 usable groups（`service/group.go:10` 自动并入），建令牌与计费两处校验都过，且不泄露折扣给他人 |
| 设额度/分组 | `PUT /api/user/` 且**必须带 role=1**（`controller/user.go:872`，否则降为游客） |
| 批发分组倍率 | option key `"GroupRatio"`，先 GET 合并再 PUT（`model/option.go:166`） |
| 建令牌 | `POST /api/token/` 需以批发账户本人登录；key 另调 `GET /api/token/{id}/key` |
| 回指渠道 | `POST /api/channel/` type=1，base_url=主站内网，自动建 ability |
| 副站建库/建 root | 容器首启 `migrateDB()` 自动建表 + 默认 `root/123456`（开通即改） |
| nginx 流式 | 模板含 `proxy_buffering off` + 长超时，避免 SSE 被缓冲卡住 |

## 注意 / 红线

- `CRYPTO_SECRET` 每副站唯一且**务必存档**（在 `${RESELLERS_DIR}/<name>/site.env`，加密库内批发 token）。
- 批发令牌已设 `model_limits`，只放开 `MODELS` 内模型——防代理商白嫖未授权模型。
- 副站默认 SQLite，零额外 DB 依赖；高并发副站改 Postgres（compose 模板内有注释）。
- `reseller.env` / `site.env` 含机密，已被 `.gitignore` 排除，切勿入库。
- 项目 Rule 5：脚本/模板/镜像/文档中的 **new-api** 与 **QuantumNous** 标识不可改删。
