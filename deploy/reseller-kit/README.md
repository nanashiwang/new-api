# reseller-kit —— 代理商副站「一键开通」脚本包

> 配套规划：`docs/reseller-distribution-plan.md`（方案 C）
> 形态：运维在**主站同一台服务器**上「复制即开通」一个代理商副站。零改 new-api 核心代码、不挂 docker.sock。

---

## 🚀 三步上手

```bash
# ① 一次性安装（装依赖 + 配置向导 + 注册全局命令）
cd <你的 newapi 仓库>/deploy/reseller-kit
sudo bash setup.sh

# ② 一次性：申请泛域名证书（计费用 1:1 原价，无需改渠道/建分组）
#    sudo certbot certonly --dns-<你的DNS商> -d "*.你的域名" -d "你的域名"

# ③ 开通一个代理商 —— 就这一行 👇（首次会自动启用 auto 分组）
sudo newapi-reseller acme acme.你的域名 20
```

参数：`newapi-reseller <name> <subdomain> [预付美元]`，最后一个可省略（用向导里设的默认值）。

---

## 它做什么

一行命令开通一个代理商：起独立副站容器（独立库/独立密钥）→ 配 nginx 子域名 → 在主站建好「批发账户(放标准客户组) + auto 令牌」→ 在副站建好「回指主站」渠道 → 改默认密码。

计费模型（1:1 复刻原价，复用现有分组，不另设折扣分组）：

```
下游用户 ─付费(零售价,副站自定)→ 副站 ─auto令牌→ 主站 ─按各模型所属分组的原价计费→ 各原厂
                                          主站从「批发账户钱包」扣额度，耗尽即背靠背断流
代理商利润 = 副站零售价 − 主站原价   （main 仍赚它本来的利润）
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

### 计费：1:1 复刻原价（无需改渠道）

代理商按你的**原价**用你**现有的分组**，所以**不用**给渠道加分组、**不用**建折扣分组：

- 批发账户放在 `RESELLER_BILLING_GROUP`（默认 `default`=你的标准客户组）→ 代理商付的就是该分组用户的同价。
- 批发令牌 `group=auto` → 按请求模型在 `AUTO_GROUPS` 内**自动选对应分组、按各自原价计费**（claude 还是 claude 价、gpt 还是 gpt 价，结构原样保留）。
- 首次开通时脚本**自动**把 `auto` 加入 `UserUsableGroups`、把 `AUTO_GROUPS` 并入 `AutoGroups`（在 `reseller.env` 里配 `AUTO_GROUPS`）。
- 代理商靠**副站零售加价**赚钱；想给批发折扣，再单独用 `GroupGroupRatio`（见规划文档），本 kit 默认不打折。

> 用 `AUTO_GROUPS` 控制开放哪些分组、用批发令牌的 `model_limits`(=`MODELS`) 控制开放哪些模型，双保险。

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
| 1:1 原价计费 | 批发令牌 `group=auto`，按模型在 `AutoGroups` 内自动选分组、按该分组原价计费（`service/group.go` `GetUserAutoGroup`/`GetUserGroupRatio`）；账户放标准客户组→付同价 |
| 设额度/分组 | `PUT /api/user/` 且**必须带 role=1**（`controller/user.go:872`，否则降为游客） |
| 启用 auto | 把 `auto` 并入 `UserUsableGroups`、开放分组并入 `AutoGroups`（option PUT，`model/option.go:124/168`） |
| 建令牌 | `POST /api/token/` 需以批发账户本人登录；key 另调 `GET /api/token/{id}/key` |
| 回指渠道 | `POST /api/channel/` type=1，base_url=主站内网，自动建 ability |
| 副站建库/建 root | 容器首启 `migrateDB()` 自动建表 + 默认 `root/123456`（开通即改） |
| nginx 流式 | 模板含 `proxy_buffering off` + 长超时，避免 SSE 被缓冲卡住 |

## 注意 / 红线

- `CRYPTO_SECRET` 每副站唯一且**务必存档**（在 `${RESELLERS_DIR}/<name>/site.env`，加密库内批发 token）。
- 批发令牌：`MODELS` 非空时设 `model_limits` 只放开这些模型；**`MODELS` 留空=全部**（关掉令牌限制，并自动从主站 `/v1/models` 拉全量给副站渠道）。
- 副站默认 SQLite，零额外 DB 依赖；高并发副站改 Postgres（compose 模板内有注释）。
- `reseller.env` / `site.env` 含机密，已被 `.gitignore` 排除，切勿入库。
- 项目 Rule 5：脚本/模板/镜像/文档中的 **new-api** 与 **QuantumNous** 标识不可改删。
