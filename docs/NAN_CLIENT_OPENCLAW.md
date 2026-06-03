# OpenClaw 配置教程

> 来自飞书《新平台配置openclaw教程》。OpenClaw 有本地网关和安全风险，使用前确认你理解本地服务、端口暴露和 API Key 泄露风险。

## 1. 平台准备

1. 登录 `https://cn.meta-api.vip/`。
2. 进入 `令牌管理` 创建令牌。
3. Claude 模型选 `claude` 分组；Codex/OpenAI 模型不要选 `claude`。
4. 可用模型以 `https://cn.meta-api.vip/pricing` 为准。
5. 复制完整 API Key。

## 2. 安装前准备

### 2.1 Node.js 和 Git

Git 下载地址：`https://git-scm.cn/`。

```bash
node --version
npm --version
```

没有 Node.js 时，去 `https://nodejs.org/` 下载 LTS。

### 2.2 环境变量

也可以把 API Key 明文写进配置文件，但更建议用环境变量。

Windows：

```bat
setx OPI_AUTH_TOKEN "你的key"
```

macOS zsh：

```bash
echo 'export OPI_AUTH_TOKEN="你的key"' >> ~/.zshrc
source ~/.zshrc
```

Linux bash：

```bash
echo 'export OPI_AUTH_TOKEN="你的key"' >> ~/.bashrc
source ~/.bashrc
```

## 3. 安装 OpenClaw

Windows 用 CMD 或 PowerShell；权限不足时用管理员运行。

```bash
npm i -g openclaw
openclaw onboard
```

onboard 时建议：

- 认真阅读安全风险提示。
- 选择快速开始配置。
- 第三方聊天平台按需配置，不需要可以跳过。
- 服务模式可选；担心风险时先不要安装服务，用前台运行。

## 4. 配置文件位置

| 系统    | 路径                                             |
| ------- | ------------------------------------------------ |
| Windows | `C:\Users\Administrator\.openclaw\openclaw.json` |
| macOS   | `~/.openclaw/openclaw.json`                      |
| Linux   | `/home/你的用户名/.openclaw/openclaw.json`       |

不要整份覆盖已有复杂配置；只替换或合并差异字段。

## 5. 单独配置 Codex / OpenAI Responses

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
        "apiKey": "你的key",
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

## 6. 单独配置 Opus / Claude

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
        "baseUrl": "https://nan.meta-api.vip/",
        "api": "anthropic-messages",
        "auth": "api-key",
        "authHeader": true,
        "apiKey": "你的key",
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

## 7. Codex 和 Opus 合并配置

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
        "apiKey": "你的key",
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
        "baseUrl": "https://nan.meta-api.vip/",
        "api": "anthropic-messages",
        "auth": "api-key",
        "authHeader": true,
        "apiKey": "你的key",
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

- `input`：模型支持的输入模态。
- `cost`：仅用于成本统计/显示，不影响请求。
- `thinkingDefault`：默认思考强度，Codex 示例为 `xhigh`。

## 8. 启动和检查

服务模式：

```bash
openclaw gateway stop
openclaw gateway start
openclaw gateway restart
```

如果提示 `Scheduled Task not installed`，先安装服务：

```bash
openclaw gateway install
openclaw gateway start
```

前台运行：

```bash
openclaw gateway
# 或
openclaw gateway run
```

检查状态：

```bash
openclaw status
```

看到 `Gateway ... reachable` 才算成功。

## 9. 常用命令

```bash
openclaw status
openclaw logs --follow
openclaw dashboard --no-open
openclaw tui
```

| 方式    | 命令                           | 说明       |
| ------- | ------------------------------ | ---------- |
| WebChat | `openclaw dashboard --no-open` | 浏览器页面 |
| CLI     | `openclaw tui`                 | 终端聊天   |

## 10. 常见问题

| 问题                            | 处理                                          |
| ------------------------------- | --------------------------------------------- |
| 权限不足                        | Windows 用管理员命令行                        |
| `gateway not connected`         | 先执行 `openclaw status`                      |
| `LLM request timed out`         | 关闭代理后重试，检查网络和防火墙              |
| `400 Instructions are required` | 确认 OpenClaw 已更新，配置是 Responses 新格式 |
| 配置脚本失败                    | 手动合并 `openclaw.json`，不要覆盖多模型配置  |
