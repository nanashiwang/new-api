# new-api 代理商 / 分销（Reseller）功能 — 完整规划方案

> 版本：v1.0 ｜ 日期：2026-06-24
> 适用项目：new-api（QuantumNous）— AI API 网关
> 本文档基于对现有代码的真实探查得出，凡涉及改造点均点名具体文件 / 表 / API。

---

## 0. 目标与需求

把当前单租户的 new-api，扩展成可「发展代理商」的分销体系：

1. **一键开通**：在本服务器上快速给代理商起一个独立站点（副站）。
2. **独立子域名**：每个代理商一个子域名（如 `acme.你的域名.com`）。
3. **配置继承 + 自动同步**：副站默认继承主站的模型、价格、品牌等；主站更新后可同步。
4. **同机隔离运行**：副站与主站跑在同一台服务器上，互不干扰。
5. **代理商自定义零售价**：代理商可设自己的模型价格。
6. **退出可迁移**：代理商不干了时，能把其名下用户（连同余额）迁回主站。

---

## 1. 核心架构决策：方案 C

### 1.1 一个决定性约束

new-api 的配置是**进程级全局单例**：

- 所有配置加载进 `common.OptionMap`（`map[string]string`）与一批 `setting.*` 包级全局变量；
- `model/option.go:244-603` 的 `updateOptionMap` 是把值写进全局变量的巨型 switch；
- `option` 表只有 `Key/Value` 两列，**无任何租户维度**；
- 价格倍率 `groupRatioMap` / `modelRatioMap` 都是全局 `RWMap`。

→ 想在「同一进程内」给不同代理商用不同配置/价格/收款，必须把配置层、计费层、倍率层全部租户化——等于把单租户系统重写成多租户系统。**这条事实直接否决了「同进程逻辑隔离」。**

### 1.2 选定方案 C —— 代理商独立实例 + 上游回指主站

> **把「多租户」降维成「代理商 = 主站的一个批发上游客户 + 自己再开一套独立 new-api 店」。**

```
下游用户 ──付费(零售价)──► 代理商副站(独立容器 + 独立库)
                              │ 上游渠道(OpenAI 兼容, 自定义 BaseURL)
                              │ base_url = 主站(内网 http://new-api:3000), key = 批发token
                              ▼
                          主站 ──付费(批发价)──► 各原厂(OpenAI/Claude/Gemini...)
```

new-api 本就极擅长「接上游、做分组定价、转发计费」，方案 C 正是复用这套能力，**核心 relay / quota / auth / payment 代码近乎零改动**。

### 1.3 方案对比（为什么是 C）

| 方案 | 思路 | 核心改造量 | 隔离强度 | 分润模型 | 契合度 |
|---|---|---|---|---|---|
| A 单实例多租户 | 同进程 + `reseller_id` | 极大（动 option/quota/ratio/auth/payment 全核心层） | 弱 | 需新建分账引擎 | ❌ 与现状对撞 |
| B 独立实例物理隔离 | 每代理商一容器+独立库 | 小 | 极强 | 缺「主站怎么赚差价」 | ✅ 高 |
| **C 独立实例 + 上游回指主站** | B + 上游回指主站赚差价 | 小-中 | 极强 | **天生清晰** | ✅✅ 最高 |

---

## 2. 三层定价与分润模型

> ⚠️ 实现更新（2026-06）：本节最初设计的「扁平批发分组 reseller_wholesale + 统一折扣」在「不同模型挂不同分组、分组倍率不同」的真实站点会算错价（claude=1.6 等加价被绕过）。`deploy/reseller-kit` 已改为 **1:1 复刻原价**：批发账户放标准客户组、批发令牌 `group=auto`，按模型在 `AutoGroups` 内自动选分组、按各自原价计费；代理商靠副站零售加价获利。需给批发折扣时再用 `GroupGroupRatio`。详见 `deploy/reseller-kit/README.md`。

三层价格全部复用现有公式 `model_ratio × group_ratio`（`service/quota.go` 已验证）：

```
① 主站批发价(主站向代理商收):
   wholesale = 主站.model_ratio[m] × 主站.GetGroupRatio("acme_wholesale")
   ──► 给批发分组配一个 <1 的折扣倍率(如 0.7)即批发七折
   ──► 落点: setting/ratio_setting/group_ratio.go (UpdateGroupRatioByJSONString)

② 代理商零售价(代理商向下游收):
   retail = 代理商副站.model_ratio[m] × 代理商副站.GetGroupRatio(下游用户group)
   ──► 完全在副站本地, 主站不可见, 代理商自由定价

③ 用户实付: 走副站标准计费链(service/quota.go), 无改动

分润: 代理商毛利 = Σ(零售消费) − Σ(批发消费)
   零售消费 = 副站库 log 表聚合
   批发消费 = 主站库 log 表 WHERE user_id = 该代理商批发账户
   两者各算各的, 无需跨库 join
```

**风控天然背靠背**：代理商批发额度耗尽 → 主站对其 token 返 402 → 副站自动对下游报错，主站不垫资。

### 2.4 批发额度怎么扣（账户与令牌如何设置）

> **由你（主站超管）在主站为代理商建「批发账户 + 批发令牌」，不是副站站长来建。** 一键开通脚本第 5–8 步自动完成这一侧。定价权与额度发放权必须握在主站，副站站长不进主站后台。

三层令牌，各管一段：

```
下游用户 ──[下游令牌]──► 副站 ──[批发令牌]──► 主站(从"批发账户"扣额度)
        (在副站里建)         (你在主站建,
                             = 副站回指渠道的 key)
```

- **批发账户**（主站的一个 User）：代表代理商，持批发余额、归入批发分组。
- **批发令牌**（该 User 的一个 token）：其 key 即副站「回指主站」渠道的 key。
- 副站 root（开通时自动建）：副站站长登录管理自己的站，与主站无关。
- 下游用户令牌：下游用户在副站建，调副站用。

扣费机制（已验证 `service/quota.go`）：

- **扣费公式**：副站每转发一次请求到主站，主站按 `批发价 = model_ratio × GetGroupRatio("x_wholesale")` 扣额度。
- **两道闸 + 双扣**：预扣同时校验 `user.quota`（`quota.go:143`）与 `token.remain_quota`（`:147`，非 unlimited 时）；后扣 `PostConsumeQuota` 同时减用户钱包（`:557`）和令牌（`:568`）。令牌 `unlimited_quota=true` 时令牌不卡不减 remain，额度完全由钱包兜底（`token.go:654`）。

推荐配置（配法 A —— 钱包做批发余额）：

| 配置项 | 值 | 说明 |
|---|---|---|
| 批发账户 `user.quota` | = 代理商预付批发额度（500000 quota = $1） | 唯一的批发钱袋 |
| 批发账户分组 | `x_wholesale`，group_ratio=0.7 | 批发折扣 = 主站差价 |
| 批发令牌 `unlimited_quota` | true | 令牌不另设上限，免维护两个额度 |
| 批发令牌 `model_limit` | 仅放开允许转售的模型 | 防越权白嫖 |

- 续费批发 = 给批发账户 `user.quota` 充值（主站 topup / 管理员加额度）。
- 耗尽 → 主站返「user quota is not enough」→ 副站背靠背报错，主站不垫资。
- 不用「令牌有限额度」配法的原因：两道闸 + 双扣下，令牌设有限额度需同时维护 `user.quota` 与 `token.remain_quota` 两个数；令牌设 unlimited 后额度统一由钱包管，最简洁。

---

## 3. 部署形态：同机多容器

### 3.1 拓扑

```
                  宿主机(你的服务器) 一张公网 IP
                              │
              ┌──────────────┴──────────────┐
              │  Caddy 反代容器(只它占 80/443) │  ← 自动 HTTPS, 按 Host 分流
              └──────────────┬──────────────┘
       ┌──────────────┬──────┴───────┬───────────────┐
  你的主域名.com   acme.你的域名   globex.你的域名      (docker 内网 newapi-net)
       │              │              │
  ┌────▼────┐   ┌─────▼─────┐  ┌─────▼─────┐
  │ 主站容器 │   │副站A容器  │  │副站B容器  │   ← 副站不对宿主暴露端口, 只 Caddy 可访问
  │ :3000   │   │ :3000     │  │ :3000     │
  └────┬────┘   └─────┬─────┘  └─────┬─────┘
       │ db=new-api   │ db=reseller_a│ db=reseller_b
       └──────────────┴──────────────┴──► Postgres(一个实例, 多个独立 database)
                                          副站回指渠道 base_url = http://new-api:3000 (走内网, 不出公网)
```

**优化**：副站「回指主站」填**内网地址** `http://new-api:3000`，调用不出公网、零延迟、不耗带宽。

### 3.2 三层隔离

1. **进程 / 文件隔离** → 容器：各自内存、`/data` 卷、配置（规避了配置进程级单例问题，因为根本不是同一进程）。
2. **数据隔离** → 独立数据库。
3. **配置隔离** → 各容器独立 env + 各自库的 `option` 表。

### 3.3 每个实例的隔离要素（全部由环境变量控制，已验证）

| 环境变量 | 作用 | 副站要点 |
|---|---|---|
| `SQL_DSN` | 数据库连接；空则 SQLite（`model/main.go:144`） | 每副站独立 database |
| `SQLITE_PATH` | SQLite 文件（`common/init.go:76`） | 每副站独立文件 |
| `REDIS_CONN_STRING` | Redis；空则纯内存（`common/redis.go:25`） | 独立库或留空 |
| `SESSION_SECRET` | Cookie 签名（`common/init.go:61`） | **每副站唯一随机串** |
| `CRYPTO_SECRET` | 加密库内渠道密钥（`common/init.go:71`） | **每副站唯一且存档** |
| `PORT` | 监听端口（`main.go:202`，默认 3000） | 容器内固定 3000，不暴露宿主 |

- 镜像统一 `calciumion/new-api:latest`（`docker-compose.yml`），保证版本一致。
- 容器首启自动建表迁移（`model/main.go` `migrateDB()` AutoMigrate）、自动建 root（`model/main.go:72-93`，开通后必须强制改密）。

### 3.4 数据库隔离选择

| 方案 | 隔离 | 资源 | 适用 |
|---|---|---|---|
| **共享 PG，独立 database + 独立账号**（推荐） | 强（账号级） | 省 | 多数副站 |
| 每副站独立 SQLite 文件 | 中 | 最省 | 小流量副站 |
| 每副站独立 Postgres 容器 | 最强 | 重 | 强隔离大客户 |

推荐共享 PG 多 database；用独立 DB 账号确保副站碰不到主站库。

---

## 4. 一键开通：快接站脚本包（MVP 形态）

### 4.1 形态决策：脚本包，而非 app 内按钮

new-api 本身**完全无宿主命令 / Docker 编排能力**（全仓库无 `os/exec` 业务调用、无 Docker SDK）。两条路：

- **app 内按钮** → 必须引入编排层，且**绝不能把 `docker.sock` 挂进业务容器**（业界公认反模式，等于宿主 root 沦陷），要额外做 provision-agent / 队列 / 状态机。复杂、慢。
- **快接站脚本包（推荐 MVP）** → 把「开通」做成运维在宿主上跑的一键脚本，**零 docker.sock 风险、几乎不改 Go 代码、几天可跑通**。它就是未来那个按钮的内核，不浪费。

> 取舍：脚本是**运维手动跑**（你 SSH 执行），非代理商自助开站。对「你主动发展代理商」的场景这恰好够用且更安全。要做自助开站再升级为 app 按钮（见阶段 2）。

### 4.2 脚本能调通的依据（已验证）

- 管理 API 鉴权：`middleware/auth.go:33` `authHelper` 从 `Authorization` 头取 `access_token`，`ValidateAccessToken`（`user.go:1755`）校验；用户 `AccessToken` 字段 char(32) uniqueIndex（`user.go:79`，注释 "this token is for system management"）。
- 开通要调的端点都在：
  - 建批发用户 `POST /api/user/`（AdminAuth，`api-router.go:179`）
  - 设角色/额度/分组 `POST /api/user/manage`（`:180`）
  - 改批发倍率 `PUT /api/option/`（RootAuth，`:233`）
  - 建回指渠道 `POST /api/channel/`（AdminAuth）

→ 脚本用「admin 的 access_token + curl」做完 app 层配置，宿主层用 shell/docker/caddy。

### 4.3 快接站包结构

```
reseller-kit/
  compose.template.yml     # 副站容器模板(端口/卷/env 占位符)
  .env.template            # 占位符: SESSION_SECRET/CRYPTO_SECRET/SQL_DSN...
  caddy-site.template      # 反代站点模板(子域名 → 容器, 自动 TLS)
  provision.sh             # 一键开通
  teardown.sh              # 一键下线(退出迁移时用)
  reseller.env             # 你存 admin access_token、主站地址等机密(chmod 600, 不进 git)
```

### 4.4 `provision.sh <name> <subdomain>` 执行步骤

```
宿主层(shell/docker/caddy):
  1. 生成唯一 SESSION_SECRET / CRYPTO_SECRET, 挑空闲端口
  2. 渲染 compose+env, 建库(PG 建 database / 或独立 SQLite 目录)
  3. docker compose -p reseller-<name> up -d   # 首启自动建表(migrateDB)+建 root
  4. 追加 Caddy 站点并 reload                   # 子域名自动签证书

app 层(curl + access_token):
  5. 主站建批发用户 → POST /api/user/ → /api/user/manage 设 role=50 / 分组 / 额度
  6. 主站写批发折扣 → PUT /api/option/ 更新 group_ratio {"<name>_wholesale": 0.7}
  7. 主站给批发用户建大额度 token(带 model_limit 防越权)
  8. 副站用其 root 凭证 → POST /api/channel/ 建「回指主站」渠道
     (Type=OpenAI 兼容, BaseURL=主站内网, Key=批发token, Models=允许集)
  9. 副站(可选)拉主站 /api/pricing 做零售价初始化
 10. 打印: 副站管理地址 + 初始管理员账号密码(强制首次改密)
```

### 4.5 关键落地细节

1. **子域名→容器 映射就记在 Caddy 站点文件里**（不必新建数据库表）。退出迁移时改这一行即可无感切换（见 §6.3）。
2. **副站 root 凭证**：第 8 步先用副站自动建的 root（`root/123456`）`POST /api/user/login` 拿凭证再调 AddChannel；开通末尾强制改默认密码。
3. **admin access_token 妥善存**：`reseller.env`、`chmod 600`、不进 git。它等于主站超管权限。
4. **批发 token 设 `model_limit`**：只放开允许转售的模型，防代理商白嫖未授权模型。
5. **`CRYPTO_SECRET` 每站唯一且存档**：丢了副站渠道密钥解不开。
6. **资源限制**：每副站容器加 `deploy.resources.limits`（cpus/mem），防一个副站打满 CPU 拖垮全机。
7. **副站不暴露宿主端口**：只 Caddy 占 80/443，副站只能经反代访问，缩小攻击面。

---

## 5. 配置 / 价格同步

### 5.1 原则

物理隔离下**默认不自动同步**（这是特性——否则会冲掉代理商自定义零售价）。模型是：

> 主站变动 → 代理商收到提示 → 代理商点按钮预览差异 → 自主选择是否应用。永远 pull + 人工确认，绝不静默覆盖。

### 5.2 new-api 已内置的「同步按钮」（现成）

| 端点 | 作用 | 行为 |
|---|---|---|
| `POST /api/ratio_sync/fetch`（`controller/ratio_sync.go` `FetchUpstreamRatios`） | 价格/倍率同步 | 拉主站 `/api/pricing`，逐模型逐字段算 **diff**，**只预览不落地**，前端勾选后才保存 |
| `POST /api/channel/upstream_updates/detect` + `/apply` | 模型列表增删 | 检测上游 `/v1/models` 新增/下架并应用 |
| `POST /api/models/sync_upstream`（带 `/preview`） | 模型元数据同步 | 预览 + 应用 |

`ratio_sync` 同步字段（`ratio_sync.go:64`）：`model_ratio / completion_ratio / cache_ratio / create_cache_ratio / image_ratio / audio_ratio / model_price / billing_mode / billing_expr`。**注意 `group_ratio` 不在其中**——代理商的分组零售加价不会被主站覆盖。
前提：主站需开启 `ExposeRatioEnabled`（默认关），否则 `/api/pricing` 不暴露倍率。

### 5.3 主站「变动」分 5 类，传播方式不同

| 变动类型 | 是否自动 | 传播方式 | 会否冲掉代理商自定义 |
|---|---|---|---|
| **批发价**（主站调 `group_ratio["x_wholesale"]`） | ✅ **自动、实时、不可拒绝** | 代理商下次请求即按新批发价扣费（`service/quota.go`） | 不涉及配置，但改变其毛利 → **必须通知** |
| 新模型上线 | 半自动（点按钮） | 渠道上游模型检测 + 倍率同步 | 否（新模型本地无价，diff 显示供采纳） |
| 模型基础倍率调整 | 半自动（点按钮） | 倍率同步 diff 表 | 否（自主决定是否跟随） |
| 站点配置 / 功能开关 / 文案 | ❌ 不自动 | 物理隔离独立；要批量推需控制平面「配置推送」 | — |
| 代码 / 镜像版本升级 | ❌ 不自动 | 重新部署容器 + DB 迁移 | — |

⚠️ **最易踩的认知坑**：「批发价」是唯一自动且不可拒绝的（它是代理商**成本**而非配置）。主站调批发价时**必须配套通知**，否则代理商困惑「我没动价，利润却缩水」。

### 5.4 推荐同步设计

```
① 开通时(一次性种子): 脚本拉主站 /api/pricing 写入副站零售价基线
② 运行时·价格与模型: 代理商后台 → 倍率同步, 上游选「主站渠道」→ 拉 diff → 勾选应用
③ 运行时·主站通知(阶段2): 主站新增模型/调批发价 → 向各副站推一条公告
④ 可选·自动跟随开关: 副站「自动跟随主站基础倍率」定时拉取并 apply 白名单字段(绝不碰 group_ratio), 默认关
```

---

## 6. 代理商退出 → 用户迁移回主站

### 6.1 可行性结论

**技术可行，但属「专项工程」，不是一键脚本能稳妥覆盖。** 副站与主站是两个独立 GORM 库，迁移本质是跨库搬运身份+余额+凭证并重定向流量。共有 **22 张表**通过 `user_id` 引用用户。前提是处理干净三个坑。

### 6.2 三个真正的坑

**坑 1：身份冲突**
`User.Username` 全局唯一（`user.go:66`），撞名直接 INSERT 失败；`Email` 非唯一（`user.go:72`）；`User.Id` 自增主键跨库必然重叠。
→ 对策：撞名改名 `<name>__<原名>`；**不保留原 id**，主站重分配并维护 `old_uid→new_uid` 映射重写所有外键；email 撞库**绝不自动合并**，列入人工复核。

**坑 2：余额折算与负债归属**
`quota` 是与价格无关的原始信用点（`user.go:80`），购买力 = `QuotaPerUnit(默认 500000=$1, common/constants.go:25) × 模型倍率 × group_ratio`。副站零售价 ≠ 主站价，同一 quota 数值两边购买力不同。
- 公平原则 = 保持「剩余能调多少 token」不变，而非保持 quota 数字不变。
- **负债归属（取决于收款模式）**：
  - **模式一（主站只收批发款）**：用户充的钱进了代理商口袋，主站从没收到这笔零售款 → 用户余额对主站是**净新增负债**。默认**不无条件转嫁主站**（代理商补差，或主站只承接已付过批发成本的部分）。
  - **模式二（主站统一收款分润）**：钱本就过主站账，承接负债顺理成章，按公式平移。
- → 迁移工具出**对账单**（原 quota / 折算 quota / 美元价值 / 负债归属），经主站 Root **审批后才写余额**，杜绝凭空生成余额。

**坑 3：用户无感知（旧 key / 旧 base_url 继续可用）** — 最优雅的部分
关键事实链：`token.key` 虽全局唯一（`token.go:21`），但主站和副站是**不同库**，副站 token 行可**原样（含 key）搬进主站库**而不撞车。于是：
1. 把 `User` 行 + `Token` 行（保留 key，重写 user_id）搬进主站库；
2. **把子域名 `acme.你的域名.com` 的反代目标从「副站容器」改指「主站容器」**。
→ `base_url` 不变、key 不变，用户**零改动**，请求实际落到主站。

### 6.3 两种迁移模式

| 模式 | 做法 | 优点 | 缺点 |
|---|---|---|---|
| **模式1 数据迁移（推荐主路径）** | 搬 `users/tokens/余额/待发放权益` 行进主站库 + 反代切换 | 用户真正无感 | 22 表外键一致性、冲突、负债承接，工程量大 |
| 模式2 凭证迁移（兜底） | 发等值兑换码或预建主站账号，通知改接入 | 实现简单、无外键地狱、负债显式可控 | 用户有感知，需改 base_url/key，流失高 |

**推荐：模式1 为主（达成无感）+ 模式2 作为撞名/孤立用户的人工兜底。**

要搬的表（必搬/选搬，取自探查）：

| 优先级 | 表 | 文件 | 处理 |
|---|---|---|---|
| 必搬 | `users` | `model/user.go:65` | 重写 id、撞名改名、折算 quota、`inviter_id` 重建 |
| 必搬 | `tokens` | `model/token.go:20` | 保留 key（撞则重签），重写 user_id，映射 group |
| 必搬 | `user_subscriptions` | `model/subscription.go` | plan_id 需主站先映射 |
| 必搬 | `subscription_issuance`（pending） | `model/subscription_issuance.go:31` | 未兑换权益带走 |
| 必搬 | `sellable_token_issuance`（pending） | `model/sellable_token.go` | 未生成 token 权益带走 |
| 必搬 | `user_oauth_binding` | `model/user_oauth_binding.go` | 共享 OAuth 应用会撞，去重 |
| 必搬 | `twofa` | `model/twofa.go` | 2FA 配置随用户 |
| 选搬 | `logs / topup / redemptions / aff_withdrawal / invite_commission_ledger / midjourney / tasks / checkin / benefit_change / invoice / payment_*` | 各对应文件 | 审计/统计类，按需 |

**邀请关系硬伤**：`User.InviterId`（`user.go:89`）指向本地 uid，副站邀请人多半不在主站（孤立链）→ 邀请人也在批次内则重指，否则置 0 失效。

### 6.4 分组归属策略

主站新建专用分组 **`<name>_legacy`**，用 `UpdateGroupRatioByJSONString`（`group_ratio.go`）配一个**贴近原副站零售价**的 group_ratio：既不污染主站 `default/vip` 定价，又让迁入用户购买力恒等（若 group_ratio 选得购买力相等，余额甚至无需缩放），后续可慢慢并轨标准价。

### 6.5 迁移流程编排

```
1. 导出副站数据(users/tokens/订阅/待发放权益/oauth/2fa + 选搬表)
2. dry-run 冲突检测(不写主站): 撞名/撞key/孤立邀请 → 清单 + 折算对账单
3. Root 审批对账单(含负债归属确认)
4. 写入主站(分批事务): 建 <name>_legacy 分组 → INSERT users(新id/改名/折算quota/group)
   → 维护 old_uid→new_uid → INSERT tokens(保留key) → 订阅/权益/oauth/2fa 按映射重写
5. 反代切换(无感关键): 子域名 upstream 从副站容器 → 主站容器, reload 网关
6. 停用副站: 置只读 → 灰度期 → docker compose down(走 teardown.sh) → 主站删批发 user/token/group ratio
7. 对账: 用户数/token数/quota总额/撞名结果 + 留存副站库快照作历史归档
```

---

## 7. 权限与角色

- 主站新增 **role=50（代理商）**，介于普通用户(1)与管理员(10)之间（`RoleRootUser=100`，`common/constants.go:198`）。
- 主站侧代理商的批发账户 = role=50，只能看自己批发账单，不进主站后台。
- 副站内代理商是自己的 root（role=100），但通过 `SidebarModulesAdmin` 裁掉危险 tab（系统设置 / 渠道编辑——防其改上游白嫖）。
- 前端落点：`web/src/helpers/storage.js`（加 `isReseller`）、`web/src/helpers/auth.jsx`（`ResellerRoute`）、`web/src/components/layout/SiderBar.jsx`、`web/src/pages/Setting/index.jsx`。

---

## 8. 收款归属

| 模式 | 做法 | 改造 | 推荐 |
|---|---|---|---|
| **模式一 代理商自收** | 代理商在副站填自己的 Stripe/易支付（物理隔离天然独立），再用收入在主站充批发额度 | 零改主站支付层 | **MVP 首选**，资金链清晰 |
| 模式二 主站统一代收+分账 | 用户付款进主站，记 `reseller_id` | `model/topup.go` 加 `reseller_id`、`service/payment_validation.go` 加维度、webhook 按代理商分发 | 阶段3 按需 |

MVP 走模式一，规避「支付全局单例、无 reseller_id」的难题。

---

## 9. 分阶段实施路线

| 阶段 | 开通 | 同步 | 退出迁移 | 收款 |
|---|---|---|---|---|
| **阶段0 PoC（~1天，0改码）** | 手动起第二容器+独立库+回指渠道+主站批发 group/token，跑通一笔双向扣费 | 手动验证倍率同步 diff | 手动演练一次「反代翻向主站」无感切换 | — |
| **阶段1 MVP（2-3周）** | **快接站脚本包**（§4）：`provision.sh` 半/全自动；Caddy 自动证书 | 现成倍率同步按钮（§5.2）+ 主站开 `ExposeRatioEnabled` | dry-run 冲突检测 + 对账单（不写主站）；模式2 兑换码人工迁移 | 模式一（代理商自收） |
| **阶段2（完整）** | 价格自动同步 + 主站「代理商账单」聚合 + 余额预警停服；（可选）app 内开通按钮 + provision-agent | 主站变更通知推送（§5.4 ③） | 模式1 全自动搬迁 + 外键重写 + 反代切换无感 + `<name>_legacy` 分组 + Root 审批流 | — |
| **阶段3（规模化）** | 编排迁 K8s API 建 Pod | — | 退出流水线产品化、自动对账、灰度+历史库归档 | 模式二（统一代收分账） |

---

## 10. 关键文件 / 表 / API 落点速查

| 改造 | 文件 / 端点 | 性质 |
|---|---|---|
| 批发分组折扣 | `setting/ratio_setting/group_ratio.go`（`UpdateGroupRatioByJSONString` / `GetGroupRatio`） | 配置（无需改码） |
| 批发计费公式 | `service/quota.go`（`model_ratio × group_ratio`） | 复用 |
| 批发 token | `model/token.go`（key uniqueIndex `:21`） | 复用 |
| 回指渠道 | `model/channel.go`（Type / BaseURL:35 / Key:26 / Models:39）；副站 `POST /api/channel/` | 复用 |
| 价格同步 | `controller/ratio_sync.go` `FetchUpstreamRatios`；`controller/pricing.go` `/api/pricing`；`ratio_setting/expose_ratio.go` | 复用 |
| 模型列表同步 | `/api/channel/upstream_updates/detect`+`apply`；`/api/channel/fetch_models/:id` | 复用 |
| 管理 API 鉴权 | `middleware/auth.go:33` `authHelper`；`ValidateAccessToken`（`user.go:1755`）；`user.AccessToken`（`user.go:79`） | 复用 |
| 建批发账户 | `POST /api/user/`（`api-router.go:179`）+ `POST /api/user/manage`（`:180`） | 复用 |
| 改倍率配置 | `PUT /api/option/`（RootAuth，`:233`） | 复用 |
| 副站自动建表/建root | `model/main.go` `migrateDB()` / `main.go:72-93` | 复用 |
| 迁移身份冲突 | `model/user.go:66/72/89`；`token.go:21`；`redemption.go` key | 新增脚本 |
| quota 语义 | `User.Quota`（`user.go:80`）；`QuotaPerUnit`（`common/constants.go:25`） | 参考 |
| 角色/前端限权 | `common/constants.go:198`；`web/src/helpers/storage.js`、`auth.jsx`、`SiderBar.jsx`、`pages/Setting/index.jsx` | 前端 |
| 子域名+证书 | 反代（Caddy/Traefik），**不进 Go** | 外围 |
| 收款模式二（阶段3） | `model/topup.go` 加 `reseller_id`；`service/payment_validation.go` | 可选重改 |

---

## 11. 风险清单

| 类别 | 风险 | 缓解 |
|---|---|---|
| 越权白嫖 | 代理商改上游渠道指向原厂绕开批发 | 副站锁渠道编辑；批发 token 设 `model_limit`/`channel_limit`；批发账户 role=50 无主站后台权限 |
| 配置漂移 | 「自动同步」覆盖代理商自定义零售价 | 同步只白名单 `model_ratio`/新模型，绝不同步 `group_ratio`；`ExposeRatioEnabled` 默认关 |
| 批发余额 | 代理商额度耗尽下游突然全报错 | 复用 token quota 背靠背 + 余额预警 + 自动通知 |
| 子域名证书 | 新子域名签证书限流 | Caddy 自动重试；高频用 `*.你的域名.com` 通配证书(DNS-01) |
| docker.sock | 编排权限暴露给业务容器 = 宿主沦陷 | **主业务容器永不挂 docker.sock**；编排走宿主脚本/独立 provisioner |
| 余额负债 | 模式一下用户余额是主站净新增负债 | 对账单显式标负债归属，Root 审批，默认不转嫁 |
| key 碰撞 | `token.key`/`redemption.key` 全局唯一 | 逐条预检，撞则重签并通知 |
| 外键完整性 | 22 表 user_id 外键，改 id 后引用断裂 | `old_uid→new_uid` 映射全表重写，单事务，dry-run 校验 |
| 邮箱归属 | email 撞主站他人账户 | 不自动合并，人工复核，改名+重新绑定 |
| 退款纠纷 | 用户曾向代理商充值，迁主站后要求退款 | 迁移前签权责协议，折算口径透明可追溯 |
| 运维线性膨胀 | N 容器 N 库，升级/备份/监控成本随代理商数增长 | 统一镜像+自动迁移；集中日志监控；规模上来上编排平台 |

---

## 12. 贯穿全程的红线（项目规约）

1. **主业务容器永不挂 `docker.sock`**（否决「app 容器自己起容器」）。
2. **任何写余额/quota 的迁移必经 Root 审批对账单**，杜绝凭空生成余额。
3. **数据库三库兼容**（SQLite / MySQL / PostgreSQL）—— 项目 Rule 2。
4. **所有 JSON 走 `common/json.go` 包装** —— 项目 Rule 1。
5. **受保护标识不可改删** —— 项目 Rule 5：脚本、模板、镜像名、文档中的 **new-api** 与 **QuantumNous** 品牌/署名/元数据严禁修改、删除或替换；代理商可叠加自己品牌，但不可替换底层归属。

---

## 13. 一句话总结

> 代理商 = 主站的一个批发上游客户 + 一套同机独立的 new-api 副站。开通用「复制即跑」的快接站脚本包（零 docker.sock 风险、几乎不改核心代码），同步复用 new-api 现成的倍率/模型拉取按钮（pull + 人工确认），退出用「搬库行 + 反代翻向主站」实现用户无感迁移。整套方案绕开「配置进程级单例」这一最大障碍，核心 relay/quota/auth/payment 代码近乎零改动。

---

## 附：建议的下一步

- **路径 A（先验证）**：实现阶段 0 PoC —— 手动开一个测试副站、跑通双向扣费、演练一次反代无感切换。用现有代码零改动证明模型成立。
- **路径 B（先交付工具）**：直接产出 `reseller-kit` 完整模板（`compose.template.yml` / `.env.template` / `provision.sh` / `teardown.sh` / `caddy-site.template` + 每个 curl 的真实端点与最小 payload），复制填参即可开通第一个副站。
