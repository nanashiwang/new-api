# OpenCode 配置教程

> 来自飞书《新平台OpenCode 配置教程》。

## 1. 平台准备

1. 登录 `https://cn.meta-api.vip/`。
2. 进入 `令牌管理` 创建令牌。
3. Codex/OpenAI 模型建议使用 `vip` / `svip`，不要选 `claude`。
4. 复制完整 API Key。

## 2. 安装环境

```bash
node --version
npm --version
npm install -g opencode-ai
opencode --version
```

Windows 如遇权限问题，用管理员 PowerShell；macOS/Linux 如遇权限问题：

```bash
sudo npm install -g opencode-ai
```

## 3. 配置环境变量

OpenCode 配置里使用 `CRS_OAI_KEY`。

Windows cmd：

```bat
set CRS_OAI_KEY=你的API密钥
```

macOS zsh：

```bash
echo 'export CRS_OAI_KEY="你的API密钥"' >> ~/.zshrc
source ~/.zshrc
```

Linux bash：

```bash
echo 'export CRS_OAI_KEY="你的API密钥"' >> ~/.bashrc
source ~/.bashrc
```

## 4. 配置 OpenCode

在 OpenCode 配置文件中添加：

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
            "low": {
              "reasoningEffort": "low",
              "reasoningSummary": "auto",
              "textVerbosity": "medium"
            },
            "medium": {
              "reasoningEffort": "medium",
              "reasoningSummary": "auto",
              "textVerbosity": "medium"
            },
            "high": {
              "reasoningEffort": "high",
              "reasoningSummary": "detailed",
              "textVerbosity": "medium"
            },
            "xhigh": {
              "reasoningEffort": "xhigh",
              "reasoningSummary": "detailed",
              "textVerbosity": "medium"
            }
          }
        }
      }
    }
  },
  "model": "nan/gpt-5.3-codex"
}
```

## 5. 使用

```bash
opencode
```

## 6. 常见问题

| 问题       | 处理                    |
| ---------- | ----------------------- |
| 401        | 确认 `CRS_OAI_KEY` 生效 |
| 404        | `baseURL` 必须带 `/v1`  |
| 模型不可用 | 检查模型广场和令牌分组  |
