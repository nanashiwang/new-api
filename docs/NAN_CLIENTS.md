# NAN 客户端接入与聊天快捷配置

> 本文已经把现有飞书教程整理为仓库内正文。底层开源项目仍基于 New API，项目归属、许可证、官方文档与上游链接保持不变。

## 1. 平台侧准备

所有客户端接入前，都先完成这几步：

1. 登录 `https://cn.meta-api.vip/`。
2. 进入 `控制台 -> 令牌管理`。
3. 新建令牌，按客户端选择正确分组。
4. 复制 API 密钥，并先在令牌管理中点击测试。
5. 可用模型以 `https://cn.meta-api.vip/pricing` 的模型广场为准。

分组不要乱选：

| 使用目标 | 推荐分组 | 说明 |
|----------|----------|------|
| Claude Code 调 Claude 模型 | `claude` | 使用 Anthropic / Claude Messages 兼容接口 |
| Claude Code 调 OpenAI / Codex 模型 | 不选 `claude`，选能访问目标模型的分组 | 使用平台上的 OpenAI / Codex 模型映射 |
| Codex CLI | `default` / `vip` / `svip` / 其他可用分组 | 使用 OpenAI Responses |
| OpenCode | `default` / `vip` / `svip` / 其他可用分组 | 使用 OpenAI 兼容配置 |
| OpenClaw 调 Codex 模型 | `default` / `vip` / `svip` / 其他可用分组 | 使用 OpenAI Responses |
| OpenClaw 调 Claude 模型 | `claude` | 使用 Anthropic Messages |
| Gemini CLI | `gemini` | 使用 Gemini 兼容接口 |

Base URL 边界：

| 接口类型 | Base URL | 常见工具 |
|----------|----------|----------|
| OpenAI / Responses | `https://cn.meta-api.vip/v1` | Codex、OpenCode、OpenClaw 的 Codex 配置 |
| Claude / Anthropic Messages | `https://cn.meta-api.vip` | Claude Code、OpenClaw 的 Claude 配置 |
| Gemini | `https://cn.meta-api.vip` | Gemini CLI |

如果控制台首页 `API 信息` 里有多条线路，以页面展示线路为准。

## 2. Node.js 与基础环境

这些 CLI 基本都依赖 Node.js。先安装 Node.js LTS 版本：

1. 打开 `https://nodejs.org/`。
2. 下载并安装 LTS 版本。
3. 重新打开终端，验证：

```bash
node --version
npm --version
```

如使用 OpenClaw，还建议安装 Git：

```text
https://git-scm.cn/
```

## 3. Codex CLI 配置

### 3.1 安装

```bash
npm install -g @openai/codex
```

### 3.2 设置 API 密钥

macOS / Linux zsh：

```bash
echo 'export NAN_API_KEY="你的API密钥"' >> ~/.zshrc
source ~/.zshrc
```

Linux bash：

```bash
echo 'export NAN_API_KEY="你的API密钥"' >> ~/.bashrc
source ~/.bashrc
```

Windows cmd：

```cmd
setx NAN_API_KEY "你的API密钥"
```

### 3.3 手动配置

在 `~/.codex/config.toml` 文件开头添加：

```toml
model_provider = "crs"
model = "gpt-5.3-codex"
model_reasoning_effort = "high"
disable_response_storage = true
preferred_auth_method = "apikey"

[model_providers.crs]
name = "crs"
base_url = "https://cn.meta-api.vip/v1"
wire_api = "responses"
requires_openai_auth = true
env_key = "NAN_API_KEY"
```

### 3.4 使用 `npx zcf` 一键配置

也可以用 `npx zcf` 走菜单配置：

```bash
npx zcf
```

菜单建议：

1. 选择平台：输入 `s` 切换到 Codex。
2. 首次安装输入 `1`，已有配置或更新密钥输入 `3`。
3. MCP 建议直接回车跳过，不安装。
4. 供应商配置选择“自定义”。
5. 供应商名称填写 `nan`。
6. URL 填写 `https://cn.meta-api.vip/v1`。
7. Key 填写平台生成的 API 密钥。
8. 配置完成后设为默认供应商并退出。

### 3.5 验证

```bash
codex --version
codex
```

## 4. Claude Code 使用 Claude 模型

这种模式用于 Claude Code 调 Claude 模型，令牌分组应选择 `claude`。

### 4.1 安装

```bash
npm install -g @anthropic-ai/claude-code
```

Linux / macOS 如遇权限问题，可在命令前加 `sudo`。

### 4.2 配置环境变量

Claude Code 需要两个变量：

```text
ANTHROPIC_BASE_URL=https://cn.meta-api.vip
ANTHROPIC_AUTH_TOKEN=你的API密钥
```

macOS / Linux zsh：

```bash
echo 'export ANTHROPIC_BASE_URL="https://cn.meta-api.vip"' >> ~/.zshrc
echo 'export ANTHROPIC_AUTH_TOKEN="你的API密钥"' >> ~/.zshrc
source ~/.zshrc
```

Linux bash：

```bash
echo 'export ANTHROPIC_BASE_URL="https://cn.meta-api.vip"' >> ~/.bash_profile
echo 'export ANTHROPIC_AUTH_TOKEN="你的API密钥"' >> ~/.bash_profile
source ~/.bash_profile
```

Windows PowerShell 临时设置：

```powershell
$env:ANTHROPIC_BASE_URL = "https://cn.meta-api.vip"
$env:ANTHROPIC_AUTH_TOKEN = "你的API密钥"
```

Windows PowerShell 永久设置：

```powershell
[System.Environment]::SetEnvironmentVariable("ANTHROPIC_BASE_URL", "https://cn.meta-api.vip", [System.EnvironmentVariableTarget]::User)
[System.Environment]::SetEnvironmentVariable("ANTHROPIC_AUTH_TOKEN", "你的API密钥", [System.EnvironmentVariableTarget]::User)
```

永久设置后，需要重新打开 PowerShell 窗口。

### 4.3 验证环境变量

macOS / Linux：

```bash
echo $ANTHROPIC_BASE_URL
echo $ANTHROPIC_AUTH_TOKEN
```

Windows PowerShell：

```powershell
echo $env:ANTHROPIC_BASE_URL
echo $env:ANTHROPIC_AUTH_TOKEN
```

### 4.4 可选：指定默认 Claude 模型

在 `~/.claude/settings.json` 中添加或替换：

```json
{
  "env": {
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "claude-sonnet-4-5-20250929"
  }
}
```

### 4.5 使用

先进入项目目录，再启动 Claude Code：

```bash
cd /path/to/your/project
claude
```

Windows 示例：

```powershell
cd C:\path\to\your\project
claude
```

验证安装：

```bash
claude --version
```

可选安装 VSCode / IDEA 插件，用于图形化查看代码差异。如果 VSCode 一直读取旧配置，彻底退出 VSCode 进程后再打开。

## 5. Claude Code 使用 OpenAI / Codex 模型

这种模式仍使用 Claude Code 客户端，但实际模型走平台上的 OpenAI / Codex 模型。

注意：令牌分组不要选择 `claude`，应选择能访问目标 OpenAI / Codex 模型的分组。

基础环境变量仍然是：

```bash
export ANTHROPIC_BASE_URL="https://cn.meta-api.vip"
export ANTHROPIC_AUTH_TOKEN="你的API密钥"
```

在 `~/.claude/settings.json` 中添加模型映射：

```json
{
  "env": {
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "5.1-codex-max",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "gpt-5.3-codex",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "gpt-5.4"
  }
}
```

`gpt-5.4` 只是示例，实际按你的需求和模型广场可用模型调整。

启动：

```bash
claude
```

## 6. Gemini CLI 配置

### 6.1 安装

```bash
npm install -g @google/gemini-cli
```

Linux / macOS 如遇权限问题，可在命令前加 `sudo`。

### 6.2 配置环境变量

Gemini CLI 需要：

```text
GOOGLE_GEMINI_BASE_URL=https://cn.meta-api.vip
GEMINI_API_KEY=你的API密钥
GEMINI_MODEL=gemini-3.1-pro-preview
```

macOS / Linux zsh：

```bash
echo 'export GOOGLE_GEMINI_BASE_URL="https://cn.meta-api.vip"' >> ~/.zshrc
echo 'export GEMINI_API_KEY="你的API密钥"' >> ~/.zshrc
echo 'export GEMINI_MODEL="gemini-3.1-pro-preview"' >> ~/.zshrc
source ~/.zshrc
```

Linux bash：

```bash
echo 'export GOOGLE_GEMINI_BASE_URL="https://cn.meta-api.vip"' >> ~/.bash_profile
echo 'export GEMINI_API_KEY="你的API密钥"' >> ~/.bash_profile
echo 'export GEMINI_MODEL="gemini-3.1-pro-preview"' >> ~/.bash_profile
source ~/.bash_profile
```

Windows PowerShell 临时设置：

```powershell
$env:GOOGLE_GEMINI_BASE_URL = "https://cn.meta-api.vip"
$env:GEMINI_API_KEY = "你的API密钥"
$env:GEMINI_MODEL = "gemini-3.1-pro-preview"
```

Windows PowerShell 永久设置：

```powershell
[System.Environment]::SetEnvironmentVariable("GOOGLE_GEMINI_BASE_URL", "https://cn.meta-api.vip", [System.EnvironmentVariableTarget]::User)
[System.Environment]::SetEnvironmentVariable("GEMINI_API_KEY", "你的API密钥", [System.EnvironmentVariableTarget]::User)
[System.Environment]::SetEnvironmentVariable("GEMINI_MODEL", "gemini-3.1-pro-preview", [System.EnvironmentVariableTarget]::User)
```

### 6.3 验证和使用

macOS / Linux：

```bash
echo $GOOGLE_GEMINI_BASE_URL
echo $GEMINI_API_KEY
echo $GEMINI_MODEL
```

Windows PowerShell：

```powershell
echo $env:GOOGLE_GEMINI_BASE_URL
echo $env:GEMINI_API_KEY
echo $env:GEMINI_MODEL
```

启动：

```bash
gemini
```

查看版本：

```bash
gemini --version
```

也可以在 Gemini CLI 中通过 `/model` 切换模型。

## 7. OpenCode 配置

### 7.1 安装

```bash
npm install -g opencode-ai
```

macOS / Linux 如遇权限问题：

```bash
sudo npm install -g opencode-ai
```

Windows 如遇权限问题，以管理员身份运行 PowerShell。

### 7.2 配置 API 密钥

macOS / Linux zsh：

```bash
echo 'export CRS_OAI_KEY="你的API密钥"' >> ~/.zshrc
source ~/.zshrc
```

Linux bash：

```bash
echo 'export CRS_OAI_KEY="你的API密钥"' >> ~/.bashrc
source ~/.bashrc
```

Windows 当前会话：

```cmd
set CRS_OAI_KEY=你的API密钥
```

### 7.3 配置 OpenCode 模型

在 OpenCode 配置文件中配置 Codex 模型：

```json
{
  "$schema": "https://opencode.ai/config.json",
  "provider": {
    "nan": {
      "npm": "@ai-sdk/openai",
      "name": "NAN OpenAI Proxy",
      "options": {
        "baseURL": "https://cn.meta-api.vip/v1",
        "apiKey": "{env:CRS_OAI_KEY}"
      },
      "models": {
        "gpt-5.3-codex": {
          "name": "GPT-5.3-Codex",
          "limit": { "context": 272000, "output": 128000 },
          "modalities": { "input": ["text", "image"], "output": ["text"] },
          "variants": {
            "low": { "reasoningEffort": "low", "reasoningSummary": "auto", "textVerbosity": "medium" },
            "medium": { "reasoningEffort": "medium", "reasoningSummary": "auto", "textVerbosity": "medium" },
            "high": { "reasoningEffort": "high", "reasoningSummary": "detailed", "textVerbosity": "medium" },
            "xhigh": { "reasoningEffort": "xhigh", "reasoningSummary": "detailed", "textVerbosity": "medium" }
          }
        }
      }
    }
  },
  "model": "nan/gpt-5.3-codex"
}
```

### 7.4 验证和启动

```bash
opencode --version
opencode
```

## 8. OpenClaw 配置

OpenClaw 有本地网关和 WebChat，存在本地服务暴露风险。使用前确认你知道本机端口、代理软件和防火墙配置，不要把本地服务暴露到不可信网络。

### 8.1 安装前准备

确认 Node.js：

```bash
node --version
npm --version
```

如果没有 Git，可安装：

```text
https://git-scm.cn/
```

### 8.2 安装 OpenClaw

NPM 安装：

```bash
npm i -g openclaw
```

初始化：

```bash
openclaw onboard
```

配置向导建议：

1. 选择快速开始配置。
2. 没有需要的第三方聊天平台可以跳过。
3. 如果之前安装过，看到 restart / skip / reinstall 时，旧配置没问题就 restart 或 skip，不放心就 reinstall。
4. 如需复杂多模型配置，不要直接整体覆盖原 `openclaw.json`，只替换不同字段。

### 8.3 设置 `OPI_AUTH_TOKEN`

Windows PowerShell 永久设置：

```powershell
[System.Environment]::SetEnvironmentVariable("OPI_AUTH_TOKEN", "你的API密钥", [System.EnvironmentVariableTarget]::User)
[System.Environment]::GetEnvironmentVariable("OPI_AUTH_TOKEN", [System.EnvironmentVariableTarget]::User)
```

Windows cmd：

```cmd
setx OPI_AUTH_TOKEN "你的API密钥"
```

macOS zsh：

```bash
echo 'export OPI_AUTH_TOKEN="你的API密钥"' >> ~/.zshrc
source ~/.zshrc
```

Linux bash：

```bash
echo 'export OPI_AUTH_TOKEN="你的API密钥"' >> ~/.bashrc
source ~/.bashrc
```

### 8.4 配置文件位置

常见位置：

| 系统 | 配置文件 |
|------|----------|
| Windows | `C:\Users\Administrator\.openclaw\openclaw.json` |
| macOS | `~/.openclaw/openclaw.json` |
| Linux | `/home/你的用户名/.openclaw/openclaw.json` |

### 8.5 单独配置 Codex / OpenAI Responses

```json
{
  "agents": {
    "defaults": {
      "model": { "primary": "crs/gpt-5.3-codex" },
      "models": {
        "crs/gpt-5.3-codex": { "alias": "codex" }
      },
      "thinkingDefault": "xhigh"
    }
  },
  "models": {
    "mode": "replace",
    "providers": {
      "crs": {
        "baseUrl": "https://cn.meta-api.vip/v1",
        "api": "openai-responses",
        "auth": "api-key",
        "authHeader": true,
        "headers": { "User-Agent": "macos-terminal-test" },
        "apiKey": "你的API密钥",
        "models": [
          {
            "id": "gpt-5.3-codex",
            "name": "GPT-5.3-Codex",
            "api": "openai-responses",
            "reasoning": true,
            "input": ["text", "image"],
            "cost": { "input": 0, "output": 0, "cacheRead": 0, "cacheWrite": 0 }
          }
        ]
      }
    }
  }
}
```

### 8.6 单独配置 Claude / Opus

```json
{
  "agents": {
    "defaults": {
      "model": { "primary": "opus/claude-opus-4-6" },
      "models": {
        "opus/claude-opus-4-6": { "alias": "opus" }
      },
      "thinkingDefault": "high"
    }
  },
  "models": {
    "mode": "replace",
    "providers": {
      "opus": {
        "baseUrl": "https://cn.meta-api.vip",
        "api": "anthropic-messages",
        "auth": "api-key",
        "authHeader": true,
        "apiKey": "你的API密钥",
        "models": [
          {
            "id": "claude-opus-4-6",
            "name": "claude-opus-4-6",
            "api": "anthropic-messages",
            "input": ["text", "image"]
          }
        ]
      }
    }
  }
}
```

### 8.7 同时配置 Codex 和 Opus

```json
{
  "agents": {
    "defaults": {
      "model": { "primary": "opus/claude-opus-4-6" },
      "models": {
        "opus/claude-opus-4-6": { "alias": "opus" },
        "crs/gpt-5.3-codex": { "alias": "codex" }
      },
      "thinkingDefault": "high"
    }
  },
  "models": {
    "mode": "replace",
    "providers": {
      "crs": {
        "baseUrl": "https://cn.meta-api.vip/v1",
        "api": "openai-responses",
        "auth": "api-key",
        "authHeader": true,
        "headers": { "User-Agent": "macos-terminal-test" },
        "apiKey": "你的API密钥",
        "models": [
          {
            "id": "gpt-5.3-codex",
            "name": "GPT-5.3-Codex",
            "api": "openai-responses",
            "reasoning": true,
            "input": ["text", "image"],
            "cost": { "input": 0, "output": 0, "cacheRead": 0, "cacheWrite": 0 }
          }
        ]
      },
      "opus": {
        "baseUrl": "https://cn.meta-api.vip",
        "api": "anthropic-messages",
        "auth": "api-key",
        "authHeader": true,
        "apiKey": "你的API密钥",
        "models": [
          {
            "id": "claude-opus-4-6",
            "name": "claude-opus-4-6",
            "api": "anthropic-messages",
            "input": ["text", "image"]
          }
        ]
      }
    }
  }
}
```

字段说明：

| 字段 | 说明 |
|------|------|
| `baseUrl` | 平台 API 地址，OpenAI Responses 需要带 `/v1`，Claude 不带 `/v1` |
| `api` | 接口类型，例如 `openai-responses`、`anthropic-messages` |
| `authHeader` | 是否通过请求头传认证信息 |
| `models[].input` | 模型支持的输入类型，例如 `text`、`image` |
| `cost` | OpenClaw 内部成本统计显示，不影响真实请求计费 |

### 8.8 启动和使用

前台运行本地网关：

```bash
openclaw gateway
# 或
openclaw gateway run
```

服务模式：

```bash
openclaw gateway install
openclaw gateway start
```

常用命令：

```bash
openclaw gateway stop
openclaw gateway start
openclaw gateway restart
openclaw status
openclaw logs --follow
openclaw dashboard --no-open
openclaw tui
```

使用方式：

| 方式 | 命令 | 说明 |
|------|------|------|
| WebChat | `openclaw dashboard --no-open` | 打开浏览器页面使用 |
| CLI | `openclaw tui` | 在终端聊天 |

### 8.9 OpenClaw 常见问题

| 问题 | 处理方式 |
|------|----------|
| `send failed: Error: gateway not connected` | 先执行 `openclaw status`，确认 Gateway reachable |
| 提示 `Scheduled Task not installed` | 没安装服务模式，改用前台运行或执行 `openclaw gateway install` |
| `LLM request timed out` | 检查代理、网络、防火墙和上游线路 |
| `400 Instructions are required` | 检查 OpenClaw 是否更新，配置是否是 Responses 新格式 |
| 权限不足 | Windows 用管理员 PowerShell；macOS / Linux 检查 npm 全局目录权限 |
| 配置脚本失败 | 手动编辑 `openclaw.json`，不要覆盖已有多模型配置 |

## 9. CC Switch 一键导入

CC Switch 用于统一管理 Claude Code、Codex、Gemini CLI、OpenCode、OpenClaw 等工具的 Provider 配置。平台的令牌管理页可以通过 `ccswitch://` Deep Link 把当前令牌一键导入 CC Switch。

安装入口：

- 官方网站：https://ccswitch.ai/
- GitHub：https://github.com/farion1231/cc-switch

### 9.1 管理员配置

进入 `系统设置 -> 聊天设置`，添加：

```json
[
  {
    "CC Switch": "ccswitch"
  }
]
```

保存后，用户在 `令牌管理 -> 聊天` 下拉菜单中选择 `CC Switch` 即可。

平台会自动生成 Deep Link，不需要用户手动拼接：

```text
ccswitch://v1/import?resource=provider&app=codex&name=<令牌名称>&endpoint=https%3A%2F%2Fcn.meta-api.vip%2Fv1&apiKey=<API密钥>&homepage=https%3A%2F%2Fcn.meta-api.vip&enabled=true
```

### 9.2 使用边界

| 场景 | 建议 |
|------|------|
| Codex / OpenAI Responses | 可直接使用当前一键导入，默认 `app=codex` |
| Claude Code | 导入后在 CC Switch 中确认目标应用和 Base URL 是否不带 `/v1` |
| Gemini CLI | 导入后确认应用类型、Base URL 和模型名 |
| OpenCode / OpenClaw | 导入后按工具要求检查 provider 字段 |

如果浏览器没有弹出 CC Switch，检查：

1. 本机是否安装 CC Switch。
2. 浏览器是否允许打开 `ccswitch://` 协议。
3. 是否被安全软件拦截 Deep Link。

修改 Provider 后，通常需要重启终端或重新启动对应 CLI。

## 10. 平台聊天快捷方式配置

配置入口：`系统设置 -> 聊天设置`。

### 10.1 数据格式

```json
[
  {
    "应用名称": "URL模板"
  }
]
```

### 10.2 支持的占位符

| 占位符 | 替换内容 | 用途 |
|--------|----------|------|
| `{address}` | 当前服务器地址，不带 `/v1` | 普通 Web Chat / iframe 聊天页 |
| `{key}` | 当前令牌完整密钥，带 `sk-` | 需要 URL 携带 key 的聊天工具 |
| `{cherryConfig}` | Base64 后的 Cherry Studio 配置 | Cherry Studio 一键导入 |
| `{aionuiConfig}` | Base64 后的 Aion UI 配置 | Aion UI 一键导入 |
| `{deepchatConfig}` | Base64 后的 DeepChat 配置 | DeepChat 一键导入 |
| `ccswitch` | 触发 CC Switch Deep Link | CC Switch 一键导入 |

### 10.3 普通 Web Chat 示例

```json
[
  {
    "Web Chat": "https://example.com/?api={key}&base={address}"
  }
]
```

普通 URL 会出现在左侧 `聊天` 区域，并通过 iframe 打开。

### 10.4 客户端一键导入示例

```json
[
  {
    "CC Switch": "ccswitch"
  }
]
```

`ccswitch`、`fluent`、`aionui`、`deepchat` 这类客户端集成不适合 iframe，会出现在令牌管理的 `聊天` 下拉菜单中。

### 10.5 安全边界

- 不要把带 `{key}` 的链接配置给不可信第三方页面。
- iframe 聊天页适合 Web Chat，不适合本地客户端协议。
- Deep Link 只负责导入配置，不保证目标客户端一定已安装。
- `{address}` 不带 `/v1`，如客户端要求 `/v1`，需要在 URL 模板里自己拼上。

## 11. 排查清单

| 问题 | 优先检查 |
|------|----------|
| 客户端提示 401 / unauthorized | API 密钥是否完整、令牌是否启用、环境变量是否生效 |
| 模型不存在 | 分组是否正确、模型是否在模型广场可用 |
| Claude Code 不通 | `ANTHROPIC_BASE_URL` 不要带 `/v1`，令牌分组应匹配模型类型 |
| Codex 不通 | `base_url` 应带 `/v1`，`wire_api` 应为 `responses` |
| Gemini 不通 | `GOOGLE_GEMINI_BASE_URL` 不要带 `/v1`，确认 `GEMINI_MODEL` 可用 |
| OpenCode 不通 | `baseURL` 应带 `/v1`，环境变量 `CRS_OAI_KEY` 是否生效 |
| OpenClaw 不通 | `openclaw status`、`openclaw logs --follow`，检查 gateway 是否 reachable |
| CC Switch 没有弹起 | 本机是否安装 CC Switch，浏览器是否允许打开 `ccswitch://` 协议 |

## 12. 原始飞书链接备查

正文已经整理在本文中，下面链接仅用于后续核对原始截图或历史教程：

| 工具 | 原始链接 |
|------|----------|
| OpenClaw | https://scn6x5davqvt.feishu.cn/docx/CFD4d4qJSou0aKxOhvHcJnkFngf?from=from_copylink |
| Codex | https://my.feishu.cn/docx/GBH7dEbn1oQl0rxXosBcSd4Knie?from=from_copylink |
| Claude Code | https://scn6x5davqvt.feishu.cn/docx/QdNfdgLFloH0uyxyPbKcoK7Gn2e?from=from_copylink |
| Claude Code(OpenAI) | https://scn6x5davqvt.feishu.cn/docx/X2RPdxmRcojHuGxDDeRc0j9mn2f |
| Gemini | https://scn6x5davqvt.feishu.cn/docx/HBFTdIsn6oGF7hxqAYEcQxnDnRh?from=from_copylink |
| OpenCode | https://scn6x5davqvt.feishu.cn/docx/X9FRdtnkuoh0qMxzsdTcRAeRn6g?from=from_copylink |
