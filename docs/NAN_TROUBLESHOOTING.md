# NAN 常见报错排查

> 本文按“错误文本 / 现象”组织，适合新手快速定位问题。底层开源项目仍基于 New API，项目归属、许可证、官方文档与上游链接保持不变。

## 快速导航

| 模块 | 适合场景 |
|------|----------|
| [先看这 5 项](#1-先看这-5-项) | 不知道从哪里开始排查 |
| [注册与登录](#2-注册与登录) | 验证码、人机校验、邀请链接 |
| [令牌与认证](#3-令牌与认证) | 401、key 无效、环境变量不生效 |
| [模型与分组](#4-模型与分组) | model not found、分组错、模型不可用 |
| [客户端配置](#5-客户端配置) | Codex、Claude Code、Gemini、OpenCode、OpenClaw |
| [流式与网络](#6-流式与网络) | stream disconnected、timeout、EOF |
| [充值与扣费](#7-充值与扣费) | 余额不足、充值未到账、扣费疑问 |
| [发票与明细账单](#8-发票与明细账单) | 订单不可选、申请状态、邮件附件 |
| [需要联系管理员时提供什么](#9-需要联系管理员时提供什么) | 提交问题时准备信息 |

## 1. 先看这 5 项

大多数问题先按这个顺序排：

1. 令牌是否复制完整，通常应是 `sk-...`。
2. 令牌是否启用，是否过期，是否额度受限。
3. 账户余额或订阅额度是否充足。
4. 模型是否在当前分组可用。
5. Base URL 是否写对，OpenAI / Responses 通常带 `/v1`，Claude / Gemini 通常不带 `/v1`。

如果还是不行，再按下面错误文本查。

## 2. 注册与登录

| 现象 / 报错 | 常见原因 | 处理方式 |
|-------------|----------|----------|
| 收不到邮箱验证码 | 邮箱填写错误、进垃圾箱、邮件服务延迟 | 检查邮箱地址和垃圾箱，等待 1-2 分钟后重试 |
| 注册提示人机校验失败 | Turnstile token 过期或重复提交 | 刷新页面，重新完成人机校验再提交 |
| 邀请关系没有绑定 | 注册链接缺少 `aff` 参数，或不是同一页面完成注册 | 使用完整邀请链接重新注册，例如 `/register?aff=xxxx` |
| 页面提示重定向过多 | 浏览器旧 Cookie 或注册链接路径异常 | 清理当前站点 Cookie，确认链接格式为 `/register?aff=xxxx` |

## 3. 令牌与认证

### 3.1 `401 unauthorized` / `invalid api key`

常见原因：

- API 密钥复制不完整。
- 环境变量没有生效。
- 令牌被禁用或删除。
- 请求头没有写 `Authorization: Bearer sk-xxxx`。
- 客户端使用了旧 key。

处理方式：

1. 到 `令牌管理` 复制完整 key。
2. 点击令牌测试，确认平台侧可用。
3. 重新打开终端或 IDE，让环境变量生效。
4. 打印环境变量确认：

```bash
echo $NAN_API_KEY
echo $ANTHROPIC_AUTH_TOKEN
echo $GEMINI_API_KEY
echo $CRS_OAI_KEY
```

Windows PowerShell：

```powershell
echo $env:NAN_API_KEY
echo $env:ANTHROPIC_AUTH_TOKEN
echo $env:GEMINI_API_KEY
echo $env:CRS_OAI_KEY
```

### 3.2 `no auth credentials` / `missing api key`

说明客户端没有读到 key。

处理方式：

- 检查环境变量名是否和客户端要求一致。
- 设置后重启终端。
- VSCode / JetBrains 插件可能读取旧环境，彻底退出 IDE 后再打开。

## 4. 模型与分组

| 现象 / 报错 | 常见原因 | 处理方式 |
|-------------|----------|----------|
| `model not found` | 模型名写错或当前分组不可用 | 去模型广场复制模型名，检查令牌分组 |
| `model unavailable` | 模型暂时不可用或渠道异常 | 换同类模型，或联系管理员看渠道状态 |
| Claude 模型调用失败 | 令牌没选 `claude` 分组 | 创建或修改令牌分组为 `claude` |
| Gemini 模型调用失败 | 令牌没选 `gemini` 分组 | 创建或修改令牌分组为 `gemini` |
| Codex 模型调用失败 | 误选 `claude` 分组 | 改用 `default` / `vip` / `svip` 等可用分组 |

## 5. 客户端配置

### 5.1 Codex CLI

正确边界：

- `base_url = "https://cn.meta-api.vip/v1"`
- `wire_api = "responses"`
- `env_key = "NAN_API_KEY"`

常见问题：

| 报错 / 现象 | 处理方式 |
|-------------|----------|
| 请求 401 | 确认 `NAN_API_KEY` 是否生效 |
| 请求 404 | 确认 `base_url` 是否带 `/v1` |
| 模型不存在 | 确认 `model = "gpt-5.3-codex"` 是否在分组可用 |

### 5.2 Claude Code

正确边界：

- `ANTHROPIC_BASE_URL="https://cn.meta-api.vip"`
- `ANTHROPIC_AUTH_TOKEN="你的API密钥"`
- Claude 模型使用 `claude` 分组。
- OpenAI / Codex 模型不要使用 `claude` 分组。

常见问题：

| 报错 / 现象 | 处理方式 |
|-------------|----------|
| 连接失败或 404 | `ANTHROPIC_BASE_URL` 不要带 `/v1` |
| 一直读旧配置 | 重启终端、VSCode、IDE 或系统 shell |
| 模型和预期不一致 | 检查 `~/.claude/settings.json` 里的默认模型映射 |

### 5.3 Gemini CLI

正确边界：

- `GOOGLE_GEMINI_BASE_URL="https://cn.meta-api.vip"`
- `GEMINI_API_KEY="你的API密钥"`
- `GEMINI_MODEL="gemini-3.1-pro-preview"`
- 令牌分组选 `gemini`。

如果模型不可用，可在 Gemini CLI 中用 `/model` 切换。

### 5.4 OpenCode

正确边界：

- `baseURL = "https://cn.meta-api.vip/v1"`
- `apiKey = "{env:CRS_OAI_KEY}"`
- 环境变量名是 `CRS_OAI_KEY`。

### 5.5 OpenClaw

常见问题：

| 报错 / 现象 | 处理方式 |
|-------------|----------|
| `gateway not connected` | 执行 `openclaw status`，确认 Gateway reachable |
| `Scheduled Task not installed` | 没安装服务模式，改用前台运行或执行 `openclaw gateway install` |
| `400 Instructions are required` | 确认 OpenClaw 已更新，并使用 Responses 新格式配置 |
| `LLM request timed out` | 检查代理、网络、防火墙、上游线路 |
| 权限不足 | Windows 用管理员 PowerShell；macOS / Linux 检查 npm 权限 |

常用命令：

```bash
openclaw status
openclaw logs --follow
openclaw gateway restart
openclaw dashboard --no-open
openclaw tui
```

### 5.6 CC Switch

| 现象 | 处理方式 |
|------|----------|
| 点击后没有弹出 | 确认本机已安装 CC Switch |
| 浏览器拦截 | 允许打开 `ccswitch://` 协议 |
| 导入后客户端不生效 | 重启对应 CLI 或终端 |
| Claude / Gemini 配置不对 | 在 CC Switch 中确认目标应用和 Base URL 规则 |

## 6. 流式与网络

| 现象 / 报错 | 常见原因 | 处理方式 |
|-------------|----------|----------|
| `stream disconnected before completion` | 上游断流、客户端超时、网络中断 | 记录时间、模型、请求 ID，联系管理员查日志 |
| `timeout awaiting response headers` | 上游响应慢或网络超时 | 换模型/线路，或稍后重试 |
| `EOF` / `unexpected EOF` | 上游连接提前关闭 | 联系管理员查渠道日志 |
| `connection refused` | 上游地址不可达或本地网关没启动 | 检查网关、代理、防火墙和渠道状态 |
| SSE 一直断 | 客户端、代理、CDN 对长连接不稳定 | 尝试非流式请求或换线路 |

提交给管理员时，最好带：请求时间、模型、令牌名称、错误文本、是否流式、客户端名称。

## 7. 充值与扣费

| 现象 | 处理方式 |
|------|----------|
| 充值未到账 | 准备订单号、支付时间、支付金额，联系管理员补单 |
| 余额不足 | 到 `钱包管理` 充值或购买订阅 |
| 不知道为什么扣费 | 到 `使用日志` 看模型、时间、消耗和令牌 |
| 订阅没生效 | 查看订阅有效期、每日额度、分组限制 |
| 扣费看起来异常 | 截图使用日志并记录请求时间给管理员 |

## 8. 发票与明细账单

| 现象 | 常见原因 | 处理方式 |
|------|----------|----------|
| 订单不能选择 | 订单未支付成功，或已在申请中/已开具 | 只选择支付成功且未被占用的订单 |
| 驳回后想重新申请 | 驳回订单会释放 | 重新提交申请 |
| 专票提交失败 | 单位名称或税号缺失 | 补齐必填项后提交 |
| 邮箱没收到附件 | 邮箱错误、进垃圾箱、邮件发送延迟 | 检查邮箱和垃圾箱，必要时联系管理员重发 |
| 需要明细账单 PDF | 管理员审核时可选择发送明细账单附件 | 联系管理员确认是否勾选附件 |

## 9. 需要联系管理员时提供什么

为了减少来回沟通，建议一次性提供：

- 账号邮箱或用户 ID。
- 问题发生时间，精确到分钟。
- 使用的客户端：Codex、Claude Code、Gemini、OpenCode、OpenClaw、API 调用等。
- 模型名和令牌名称，不要直接发完整 API Key。
- 完整错误文本或截图，截图要遮住 API Key。
- 是否流式请求。
- 如果是支付问题，提供订单号、支付金额、支付时间。
