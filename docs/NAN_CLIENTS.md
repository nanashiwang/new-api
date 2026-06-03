# NAN 客户端接入总览

> 本页只做选择和速查；每个工具的完整步骤已经拆到独立页面。飞书教程正文已迁移到对应页面。

## 1. 我该选哪个

| 目标                             | 推荐页面                                                 | 分组                       | Base URL                     |
| -------------------------------- | -------------------------------------------------------- | -------------------------- | ---------------------------- |
| 终端里用 Codex 写代码            | [Codex](./NAN_CLIENT_CODEX.md)                           | `default` / `vip` / `svip` | `https://cn.meta-api.vip/v1` |
| Claude Code 调 Claude 模型       | [Claude Code](./NAN_CLIENT_CLAUDE_CODE.md)               | `claude`                   | `https://cn.meta-api.vip`    |
| Claude Code 调 OpenAI/Codex 模型 | [Claude Code OpenAI](./NAN_CLIENT_CLAUDE_CODE_OPENAI.md) | 非 `claude`                | `https://cn.meta-api.vip`    |
| Gemini CLI                       | [Gemini](./NAN_CLIENT_GEMINI.md)                         | `gemini`                   | `https://cn.meta-api.vip`    |
| OpenCode                         | [OpenCode](./NAN_CLIENT_OPENCODE.md)                     | `default` / `vip` / `svip` | `https://cn.meta-api.vip/v1` |
| WebChat / 本地网关               | [OpenClaw](./NAN_CLIENT_OPENCLAW.md)                     | 按模型选择                 | OpenAI 带 `/v1`，Claude 不带 |
| 一键导入配置                     | [CC Switch 和聊天入口](./NAN_CLIENT_CCSWITCH.md)         | 按工具选择                 | 平台自动生成                 |

## 2. 最容易填错的地方

| 错误                            | 正确做法                                   |
| ------------------------------- | ------------------------------------------ |
| Codex Base URL 没带 `/v1`       | 填 `https://cn.meta-api.vip/v1`            |
| Claude Code Base URL 带了 `/v1` | 填 `https://cn.meta-api.vip`               |
| Claude 模型没选 `claude` 分组   | 令牌分组选 `claude`                        |
| Gemini 模型没选 `gemini` 分组   | 令牌分组选 `gemini`                        |
| OpenAI/Codex 模型选了 `claude`  | 改成 `default` / `vip` / `svip` 等可用分组 |
| 环境变量改了不生效              | 重新打开终端、VSCode、IDE                  |

## 3. 通用准备

所有客户端都先做这 4 步：

1. 登录 `https://cn.meta-api.vip/`。
2. 进入 `令牌管理` 创建令牌。
3. 按模型选择正确分组。
4. 复制完整 API Key。

模型以 `https://cn.meta-api.vip/pricing` 展示为准，不同分组可用模型不同。

## 4. 通用环境

多数 CLI 依赖 Node.js：

```bash
node --version
npm --version
```

没有 Node.js 时，去 `https://nodejs.org/` 下载 LTS 版本。

## 5. 独立教程

- [Codex 配置教程](./NAN_CLIENT_CODEX.md)
- [Claude Code 使用 Claude 模型](./NAN_CLIENT_CLAUDE_CODE.md)
- [Claude Code 使用 OpenAI/Codex 模型](./NAN_CLIENT_CLAUDE_CODE_OPENAI.md)
- [Gemini CLI 配置教程](./NAN_CLIENT_GEMINI.md)
- [OpenCode 配置教程](./NAN_CLIENT_OPENCODE.md)
- [OpenClaw 配置教程](./NAN_CLIENT_OPENCLAW.md)
- [CC Switch 与聊天入口配置](./NAN_CLIENT_CCSWITCH.md)

## 6. 原始飞书链接备查

正文已经迁移，下面只用于核对历史版本：

| 工具                | 原始链接                                                        |
| ------------------- | --------------------------------------------------------------- |
| OpenClaw            | https://scn6x5davqvt.feishu.cn/docx/CFD4d4qJSou0aKxOhvHcJnkFngf |
| Codex               | https://my.feishu.cn/docx/GBH7dEbn1oQl0rxXosBcSd4Knie           |
| Claude Code         | https://scn6x5davqvt.feishu.cn/docx/QdNfdgLFloH0uyxyPbKcoK7Gn2e |
| Claude Code(OpenAI) | https://scn6x5davqvt.feishu.cn/docx/X2RPdxmRcojHuGxDDeRc0j9mn2f |
| Gemini              | https://scn6x5davqvt.feishu.cn/docx/HBFTdIsn6oGF7hxqAYEcQxnDnRh |
| OpenCode            | https://scn6x5davqvt.feishu.cn/docx/X9FRdtnkuoh0qMxzsdTcRAeRn6g |
