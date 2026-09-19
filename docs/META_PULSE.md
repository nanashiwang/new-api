# Meta Pulse 集成配置

Meta Pulse 是独立的增长与权益服务，不进入 new-api relay 请求主链路。中奖后，Pulse 将额度奖励自动发给已绑定的 new-api 账号，无需导入兑换码。内部奖励接口只用于内网或受控网关之间的服务调用，公网不应直接暴露。

## 用户页展示

「Meta Pulse」用户页展示等级、经验值、可用奖励券和奖励历史，不展示账本、活动周期或本期数值。经验值沿用接口的累计 `lifetime_contribution_milli`，按千分单位换算；本次为展示名称调整，既有经验数据、等级计算、后台记账及奖励结算规则不变。

## 在管理面板配置 new-api

升级到支持本配置页的版本后，进入「系统设置 → 配置 Meta Pulse 对接」。以下 new-api 配置均可在页面保存，无需为了修改这些设置重建容器：

| 配置 | 对应部署环境变量 | 用途 |
|---|---|---|
| 运行环境 | `PULSE_ENV` | 生产环境使用 `production`，执行强密钥和 Redis 防重放检查 |
| 强制保留消费日志 | `PULSE_USAGE_LOG_REQUIRED` | Pulse 依赖消费日志时保持开启，阻止关闭消费日志 |
| Pulse 内网地址 | `PULSE_INTERNAL_URL` | new-api 能访问的 Pulse 地址，例如 `http://pulse-api:8088` |
| Pulse 用户侧密钥 | `PULSE_USER_BFF_HMAC_SECRET` | 与 Pulse 同名配置一致，用于用户摘要与奖励记录 |
| Pulse 运营侧密钥 | `PULSE_ADMIN_HMAC_SECRET` | 与 Pulse 同名配置一致，用于运营概览 |
| 自动发奖密钥及轮换旧密钥 | `PULSE_SERVICE_HMAC_SECRET`、`PULSE_SERVICE_HMAC_SECRET_PREVIOUS` | 接收 Pulse Worker 的发奖和查询请求 |
| 撤销密钥及轮换旧密钥 | `PULSE_ROLLBACK_HMAC_SECRET`、`PULSE_ROLLBACK_HMAC_SECRET_PREVIOUS` | 接收 Pulse 管理 API 的查询和受控撤销请求 |
| 允许新奖励到账 | `PULSE_BENEFIT_ENABLED` | 默认关闭；关闭后历史重放、查询和受控撤销仍可使用 |
| 单笔额度上限 | `PULSE_BENEFIT_MAX_GRANT_QUOTA` | 单笔奖励的正整数 quota 上限 |
| 单用户每日额度上限 | `PULSE_BENEFIT_USER_DAILY_QUOTA` | 同一用户一天累计收到的正整数 quota 上限 |
| 全站每日额度上限 | `PULSE_BENEFIT_DAILY_QUOTA` | 全站一天累计发出的正整数 quota 上限 |
| 论坛 SSO 密钥及轮换旧密钥 | `PULSE_FORUM_SSO_SECRET`、`PULSE_FORUM_SSO_SECRET_PREVIOUS` | 与社区插件的 SSO 配置对齐 |
| 论坛 SSO 回调地址 | `PULSE_FORUM_SSO_CALLBACK_URL` | 例如 `https://metar.uk/api/user-center/login/callback` |

页面将本次修改的配置存入 new-api 数据库，数据库中的对应字段优先于部署环境变量；没有修改的字段继续沿用原有配置，未设置的字段仍回退到环境变量。已有的内网地址、用户侧密钥、运营侧密钥设置也会保留。修改同名环境变量不会覆盖已经明确保存的字段，因此后续调整这些字段仍应在页面完成。

保存后，处理该请求的实例立即使用新配置；共享数据库的其他实例按既有 `SYNC_FREQUENCY` 周期同步，默认 60 秒。多实例切换密钥或启停奖励时，须为同步留出时间。所有实例仍须先部署支持这些设置的新版本；页面配置不能代替代码升级、数据库迁移或 Redis、数据库等基础服务配置。

### 保留已有密钥与安全轮换

密钥输入框留空表示保留当前值，页面只展示已配置状态，不回传已有密钥。不要因输入框为空就重新生成整套密钥，也不要修改已有数据库密码、SSO 密钥或其他随机密钥。明确执行清除操作才会删除对应密钥；清除后不会重新启用同名环境变量中的旧值。轮换结束清除旧密钥也应使用明确清除操作。

「生成新密钥」只生成并展示新值，不自动保存，不会自动修改 Pulse 或社区配置。每个签名角色应使用不同的强随机密钥，尤其不能将发奖、撤销、用户侧、运营侧和 SSO 密钥混用。真实密钥不得提交 Git、写入日志或交给普通用户页面。

发奖与撤销由 new-api 接收验签：先在接收端设置新密钥并把旧密钥放入对应 `*_PREVIOUS`，待所有 new-api 实例同步后，再切换 Pulse 发送端，确认旧请求排空后清除旧密钥。用户侧与运营侧请求由 new-api 签名、Pulse 验签，它们的轮换旧密钥应配置在 Pulse 接收端。SSO 由 new-api 使用当前密钥签发，社区插件必须在轮换期间接受新旧密钥，之后再清除旧值。不能只改一端就认为轮换完成。

### 额度单位与启用次序

额度按整数 quota 配置。先查看 `/api/status` 的 `data.quota_per_unit`，再据此配置 Pulse 的 `PULSE_QUOTA_PER_UNIT`。例如返回 `500000` 时，单笔 `250000`、单用户每日 `1000000`、全站每日 `5000000`，分别表示 0.5、2、10 API 额度单位；这些不是人民币金额，也不是实际运营预算建议。

三个上限均须为有效正整数，才能启用新奖励到账。每日上限按北京时间自然日累计毛发放额，撤销奖励不会恢复当天限额；限制在数据库事务中检查，Redis 不是额度事实源。

先保持「允许新奖励到账」关闭，填写生产环境、消费日志保护、独立密钥和接收端限额，保存并核对各实例同步。待 Pulse 奖池、预算和社区绑定准备完成后，再进行受控的小额验收，按上线计划开启该开关与 Pulse 的新抽奖开关。生产验收应核对奖励凭证、账本和实际账户余额，不能只看前端成功提示。

## Pulse 与社区仍须分别配置

new-api 页面只管理 new-api 自身的设置，不会写入其他服务：

- Pulse 保留原有数据库密码和只读日志账号；配置 `NEWAPI_INTERNAL_BASE_URL`、与 new-api 一致的发奖/撤销密钥和额度换算单位。发奖密钥只交给 Worker，撤销密钥交给管理 API。`PULSE_ACTIONS_ENABLED`、`PULSE_REWARD_SHADOW_MODE`、奖池创建与数据库迁移仍在 Pulse 侧完成；开启抽奖不需要开启周期奖励 `PULSE_PERIOD_REWARDS_ENABLED`。
- 社区插件的 `sso_hmac_secret` 对应 new-api 的论坛 SSO 密钥；`pulse_hmac_secret` 对应 Pulse 的 `PULSE_FORUM_HMAC_SECRET`，不是自动发奖密钥。插件的 `community_bff_hmac_secret` 与 Pulse 的 `PULSE_COMMUNITY_BFF_HMAC_SECRET` 使用另一组独立值，该值不属于 new-api 配置。
- 原有绑定正常时保留已有 SSO 设置。论坛回调必须为 HTTPS，路径固定为 `/api/user-center/login/callback`，不允许用户名密码、query 或 fragment。

完整付费来源与升级要求见 [PULSE_FUNDING_PROVENANCE.md](PULSE_FUNDING_PROVENANCE.md)。新奖池、奖项概率、预算和产券门槛按 Pulse 的上线文档操作。

## 部署环境兼容与内部接口

原有部署仍可通过环境变量提供上述设置；完整示例见仓库 `.env.example`。Compose 使用环境变量时，必须把配置实际注入 new-api 容器，仅修改用于 Compose 插值的 `.env` 不保证传入容器。可用权限为 `600` 的 `pulse.env` 配合 `env_file`，并检查 `environment` 中的同名值是否覆盖它。环境变量变更需要重建容器；页面保存的配置则按前述规则即时生效和同步。

Benefit 内部接口使用已验签服务身份独立限流，默认每 60 秒 600 次，不使用公共 API 的 IP 限流。`PULSE_BENEFIT_RATE_LIMIT_ENABLE`、`PULSE_BENEFIT_RATE_LIMIT`、`PULSE_BENEFIT_RATE_LIMIT_DURATION` 仍是部署环境参数，未包含在本次页面配置范围。SSO 入口另使用 CriticalRateLimit。生产环境仍须配置 Redis 防重放，并按真实吞吐完成压测。

奖励路由前缀为 `/api/internal/pulse/benefits`：

- `POST /grant`：仅接受 `pulse-settlement` 角色，按 `source_ref` 幂等发放不可转赠额度；`grant_id` 必须等于 `source_ref`。
- `POST /query` 或 `GET /query/:source_ref`：接受 `pulse-settlement` 或 `pulse-rollback` 角色，查询原始奖励状态，用于超时恢复；不得更换 `source_ref` 重发。
- `POST /rollback`：仅接受独立的 `pulse-rollback` 角色，使用原始 `source_ref` 追加可审计的撤销记录。

Grant 的请求体 `user_id` 必须与已验签的 `X-Pulse-User-Id` 一致。同一 `source_ref` 携带相同 payload 返回幂等成功，payload、用户或额度不一致返回 conflict。奖励凭证与钱包额度入账在同一事务中完成，不增加可转赠额度或付费抽奖资格。

用户只读入口 `/api/pulse/summary` 与 `/api/pulse/rewards` 由 new-api BFF 代理，身份从 `UserAuth` 上下文派生；运营概览 `/api/pulse/ops/overview` 使用独立的 `admin` 角色密钥。签名均在服务端完成，不向 Pulse 转发浏览器 Cookie、Authorization 或用户自报身份。

论坛登录入口为 `/api/forum/sso/start`。未登录用户先回到 `/login?next=/api/forum/sso/start`，登录（含 2FA）成功后由后端签发短期、单次 Login Ticket，并重定向到已配置的固定回调地址。浏览器不能通过 query 参数声明可信 `user_id`。
