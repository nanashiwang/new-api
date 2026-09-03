# Meta Pulse 集成配置

Meta Pulse 是独立的增长与权益服务，不进入 new-api relay 请求主链路。以下接口只用于内网或受控网关之间的服务调用，公网不应直接暴露。

## 服务密钥

在 new-api 与 Pulse 之间配置同一份强随机 `PULSE_SERVICE_HMAC_SECRET`，用于奖励结算接口的 HMAC 请求签名。每个请求必须包含用户 ID、角色、时间戳、随机 nonce 和请求体摘要；生产环境需要启用 Redis，避免多实例重复使用 nonce。密钥轮换时先设置接收端的 `*_PREVIOUS`，再切换发送端到新密钥，确认旧请求排空后清空旧密钥；缺少当前密钥或 Redis 时生产环境必须拒绝请求。

用户只读入口 `/api/pulse/summary` 与 `/api/pulse/rewards` 由 new-api BFF 代理。BFF 从 `UserAuth` 上下文派生用户 ID，使用独立的 `PULSE_USER_BFF_HMAC_SECRET` 签名调用 `PULSE_INTERNAL_URL`，不会向 Pulse 转发浏览器 Cookie、Authorization 或用户自报身份。

论坛单点登录另使用 `PULSE_FORUM_SSO_SECRET`。它只由 new-api 签发短期、单次 Login Ticket，论坛插件负责验签和消费。回调地址必须是 HTTPS 且不带 fragment：

```dotenv
PULSE_ENV=production
PULSE_SERVICE_HMAC_SECRET=请注入强随机值
PULSE_SERVICE_HMAC_SECRET_PREVIOUS=轮换期间临时保留旧值
PULSE_INTERNAL_URL=http://pulse-api:8088
PULSE_USER_BFF_HMAC_SECRET=请注入另一份强随机值
PULSE_USER_BFF_HMAC_SECRET_PREVIOUS=轮换期间临时保留旧值
PULSE_FORUM_SSO_SECRET=请注入强随机值
PULSE_FORUM_SSO_SECRET_PREVIOUS=轮换期间临时保留旧值
PULSE_FORUM_SSO_CALLBACK_URL=https://forum.example.com/api/user-center/login/callback
```

不要把真实密钥提交到 Git，也不要让浏览器、论坛前端或 YuanHeng 持有服务密钥。

## Pulse 奖励接口

路由前缀为 `/api/internal/pulse/benefits`，仅接受 `pulse-settlement` 角色的服务签名：

- `POST /grant`：按 `source_ref` 幂等发放不可转赠额度；`grant_id` 必须等于 `source_ref`。
- `POST /query` 或 `GET /query/:source_ref`：查询原始奖励状态，用于超时后的恢复；不得更换 `source_ref` 重发。
- `POST /rollback`：使用原始 `source_ref` 追加可审计的撤销记录。

同一 `source_ref` 携带相同 payload 会返回幂等成功；payload、用户或额度不一致会返回 conflict。Pulse 奖励不会写入可转赠额度，也不会直接修改 `users.quota`。

## 论坛登录入口

论坛应跳转到 `/api/forum/sso/start`。new-api 从现有 session 读取用户身份：未登录时回到 `/login?next=/api/forum/sso/start`，登录（含 2FA）成功后由后端签发 Ticket 并重定向到固定回调地址。浏览器不能通过 query 参数声明可信 `user_id`。
