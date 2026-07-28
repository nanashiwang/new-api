# CC Switch 一键配置

## 用户操作

1. 安装并启动 [CC Switch](https://ccswitch.ai/)。
2. 打开平台的 `令牌管理`。
3. 在目标令牌右侧点击 `CC Switch`。
4. 选择目标应用、默认模型和 API 线路。
5. 点击 `打开 CC Switch 并自动导入`，在 CC Switch 中确认。

平台会根据该令牌当前真正可用的模型筛选选项，并自动处理不同客户端的 Base URL：

| 目标应用    | 协议要求                           | 默认选择规则                         |
| ----------- | ---------------------------------- | ------------------------------------ |
| Codex       | OpenAI Responses，地址带 `/v1`     | 优先 `gpt-5.6-sol`，不可用则安全回退 |
| Claude Code | Anthropic Messages，地址不带 `/v1` | 只显示 Anthropic 兼容模型            |
| Gemini CLI  | Gemini 原生接口，地址不带 `/v1`    | 只显示 Gemini 兼容模型               |
| OpenCode    | OpenAI 兼容接口，地址带 `/v1`      | 只显示 OpenAI 文本模型               |
| OpenClaw    | OpenAI 兼容接口，地址带 `/v1`      | 只显示 OpenAI 文本模型               |

图像、音频、OCR、嵌入、重排和视频模型不会出现在客户端配置列表中。令牌没有兼容模型时，页面会阻止导入，不会生成一个注定失败的配置。

## 安全与行为边界

- 完整密钥仅在用户点击配置时按需读取，页面不展示密钥，也不写入日志或持久化存储。
- Deep Link 只发送给本机的 `ccswitch://` 协议处理器。
- 页面不会自动发起收费的模型请求；令牌模型列表检查不产生模型用量。
- 每次导入由 CC Switch 生成 Provider。重复导入可能产生同名 Provider，用户应在 CC Switch 中确认或清理旧配置。
- 平台不会删除或覆盖用户现有的 CC Switch Provider。

## 无法打开时

1. 确认 CC Switch 已安装并正在运行。
2. 允许浏览器打开 `ccswitch://` 外部协议。
3. 检查浏览器或安全软件是否拦截外部应用。
4. 在 CC Switch 中确认导入后，重启目标 CLI 或新开一个终端。

## 管理员兼容配置

令牌行现在始终显示独立的 `CC Switch` 按钮，不再依赖聊天设置。为兼容旧入口，`系统设置 -> 聊天设置` 中原有配置仍可保留：

```json
[
  {
    "CC Switch": "ccswitch"
  }
]
```

旧入口和新按钮都会打开同一个安全配置弹窗。

## 其他聊天快捷方式

聊天设置仍支持以下占位符：

| 占位符             | 替换内容                       |
| ------------------ | ------------------------------ |
| `{address}`        | 当前服务器地址，不带 `/v1`     |
| `{key}`            | 当前令牌完整密钥               |
| `{cherryConfig}`   | Base64 后的 Cherry Studio 配置 |
| `{aionuiConfig}`   | Base64 后的 Aion UI 配置       |
| `{deepchatConfig}` | Base64 后的 DeepChat 配置      |

不要把带 `{key}` 的链接配置给不可信第三方页面。
