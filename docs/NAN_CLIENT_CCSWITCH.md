# CC Switch 与聊天入口配置

## 1. CC Switch 是什么

CC Switch 用于统一管理 Claude Code、Codex、Gemini CLI、OpenCode、OpenClaw 等工具的 Provider 配置。平台令牌管理页可通过 `ccswitch://` Deep Link 把当前令牌一键导入。

安装入口：

- 官网：https://ccswitch.ai/
- GitHub：https://github.com/farion1231/cc-switch

## 2. 管理员配置

入口：`系统设置 -> 聊天设置`。

添加：

```json
[
  {
    "CC Switch": "ccswitch"
  }
]
```

保存后，用户在 `令牌管理 -> 聊天` 下拉菜单选择 `CC Switch`。

平台会自动生成 Deep Link，不需要用户手动拼接。

示例：

```text
ccswitch://v1/import?resource=provider&app=codex&name=<令牌名称>&endpoint=https%3A%2F%2Fcn.meta-api.vip%2Fv1&apiKey=<API密钥>&homepage=https%3A%2F%2Fcn.meta-api.vip&enabled=true
```

## 3. 使用边界

| 场景                     | 建议                                  |
| ------------------------ | ------------------------------------- |
| Codex / OpenAI Responses | 可直接使用一键导入，默认 `app=codex`  |
| Claude Code              | 导入后确认 Base URL 不带 `/v1`        |
| Gemini CLI               | 导入后确认应用类型、Base URL 和模型名 |
| OpenCode / OpenClaw      | 导入后按工具要求检查 provider 字段    |

如果浏览器没有弹出 CC Switch：

1. 确认本机已安装 CC Switch。
2. 允许浏览器打开 `ccswitch://` 协议。
3. 检查安全软件是否拦截 Deep Link。
4. 修改 Provider 后，重启终端或对应 CLI。

## 4. 平台聊天快捷方式

入口：`系统设置 -> 聊天设置`。

数据格式：

```json
[
  {
    "应用名称": "URL模板"
  }
]
```

## 5. 支持占位符

| 占位符             | 替换内容                       | 用途                     |
| ------------------ | ------------------------------ | ------------------------ |
| `{address}`        | 当前服务器地址，不带 `/v1`     | Web Chat / iframe 聊天页 |
| `{key}`            | 当前令牌完整密钥               | 需要 URL 携带 key 的工具 |
| `{cherryConfig}`   | Base64 后的 Cherry Studio 配置 | Cherry Studio 一键导入   |
| `{aionuiConfig}`   | Base64 后的 Aion UI 配置       | Aion UI 一键导入         |
| `{deepchatConfig}` | Base64 后的 DeepChat 配置      | DeepChat 一键导入        |
| `ccswitch`         | 触发 CC Switch Deep Link       | CC Switch 一键导入       |

## 6. 普通 Web Chat 示例

```json
[
  {
    "Web Chat": "https://example.com/?api={key}&base={address}"
  }
]
```

普通 URL 会出现在左侧 `聊天` 区域，并通过 iframe 打开。

## 7. 安全边界

- 不要把带 `{key}` 的链接配置给不可信第三方页面。
- iframe 适合 Web Chat，不适合本地客户端协议。
- Deep Link 只负责导入配置，不保证目标客户端已安装。
- `{address}` 不带 `/v1`，如客户端要求 `/v1`，在 URL 模板里自己拼上。
