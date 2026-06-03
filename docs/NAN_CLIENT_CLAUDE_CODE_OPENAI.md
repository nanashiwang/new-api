# Claude Code 使用 OpenAI / Codex 模型

> 来自飞书《新平台ClaudeCode使用OpenAi的模型》，适合在 Claude Code 里调用 OpenAI/Codex 模型。

## 1. 平台准备

1. 登录 `https://cn.meta-api.vip/`。
2. 进入 `令牌管理` 创建令牌。
3. 使用 OpenAI/Codex 模型时，不要选择 `claude` 分组。
4. 可用模型以 `https://cn.meta-api.vip/pricing` 为准。
5. 复制完整 API Key。

## 2. 安装 Claude Code

```bash
node --version
npm --version
npm install -g @anthropic-ai/claude-code
claude --version
```

没有 Node.js 时，去 `https://nodejs.org/` 下载 LTS。

## 3. 配置环境变量

Claude Code 的 Base URL 不带 `/v1`：

```text
ANTHROPIC_BASE_URL=https://cn.meta-api.vip
ANTHROPIC_AUTH_TOKEN=你的API密钥
```

Windows 临时设置：

```powershell
$env:ANTHROPIC_BASE_URL = "https://cn.meta-api.vip"
$env:ANTHROPIC_AUTH_TOKEN = "你的API密钥"
```

Windows 永久设置：

```powershell
[System.Environment]::SetEnvironmentVariable("ANTHROPIC_BASE_URL", "https://cn.meta-api.vip", [System.EnvironmentVariableTarget]::User)
[System.Environment]::SetEnvironmentVariable("ANTHROPIC_AUTH_TOKEN", "你的API密钥", [System.EnvironmentVariableTarget]::User)
```

macOS / Linux：

```bash
export ANTHROPIC_BASE_URL="https://cn.meta-api.vip"
export ANTHROPIC_AUTH_TOKEN="你的API密钥"
```

zsh 永久设置：

```bash
echo 'export ANTHROPIC_BASE_URL="https://cn.meta-api.vip"' >> ~/.zshrc
echo 'export ANTHROPIC_AUTH_TOKEN="你的API密钥"' >> ~/.zshrc
source ~/.zshrc
```

## 4. 指定 OpenAI / Codex 模型

编辑 `~/.claude/settings.json`：

```json
{
  "env": {
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "5.1-codex-max",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "gpt-5.3-codex",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "gpt-5.4"
  }
}
```

模型可按实际需求替换，不一定必须使用 `gpt-5.4`。

## 5. 使用

```bash
claude
```

## 6. 常见问题

| 问题               | 处理                              |
| ------------------ | --------------------------------- |
| 模型不可用         | 检查模型广场和令牌分组            |
| 误选 `claude` 分组 | 改成 OpenAI/Codex 可用分组        |
| IDE 读取旧配置     | 完全退出 VSCode / IDEA 后重开     |
| 连接失败           | `ANTHROPIC_BASE_URL` 不要带 `/v1` |
