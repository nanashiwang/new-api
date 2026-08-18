# NAN 各厂商模型调用请求样例

> 本页面向第一次配置第三方客户端或直接调用 API 的用户。所有厂商统一通过 `https://cn.meta-api.vip` 调用，不需要改成厂商官网域名。
>
> MiMo TTS、音色设计、音色克隆和 ASR 的完整示例请查看 [MiMo V2.5 语音模型调用指南](./NAN_MIMO_AUDIO.md)。

模型快照更新于 2026-07-15。模型、分组和协议会随渠道调整，正式接入前请查询现网模型列表。

## 1. 第三方客户端怎么填

### 1.1 OpenAI 兼容客户端

适用于 Cherry Studio、OpenCode、Lobe Chat、ChatBox、自建程序和大多数支持自定义模型的第三方客户端。

| 配置项   | 填写内容                                |
| -------- | --------------------------------------- |
| API 类型 | OpenAI Compatible                       |
| Base URL | `https://cn.meta-api.vip/v1`            |
| API Key  | 在 `控制台 -> 令牌管理` 创建的 `sk-...` |
| 模型     | 填当前令牌实际可见的完整模型名          |

### 1.2 Claude / Anthropic 客户端

| 配置项   | 填写内容                                |
| -------- | --------------------------------------- |
| API 类型 | Anthropic / Claude                      |
| Base URL | `https://cn.meta-api.vip`               |
| API Key  | 在 `控制台 -> 令牌管理` 创建的 `sk-...` |
| 模型     | `claude-sonnet-5`、`claude-opus-4-8` 等 |

Claude Base URL 通常不要手动补 `/v1`，客户端会自动请求 `/v1/messages`。

### 1.3 Gemini 客户端

| 配置项   | 填写内容                                        |
| -------- | ----------------------------------------------- |
| API 类型 | Google Gemini                                   |
| Base URL | `https://cn.meta-api.vip`                       |
| API Key  | 在 `控制台 -> 令牌管理` 创建的 `sk-...`         |
| 模型     | `gemini-3.5-flash`、`gemini-3.1-pro-preview` 等 |

如果第三方客户端不支持 Gemini 自定义地址，可以改用 OpenAI Compatible，并通过 `/v1/chat/completions` 调 Gemini 模型。

## 2. 协议与请求地址

| 协议                    | 请求地址                                 | 适用模型                                         |
| ----------------------- | ---------------------------------------- | ------------------------------------------------ |
| OpenAI Chat Completions | `/v1/chat/completions`                   | GPT、DeepSeek、xAI，以及 Claude/Gemini 兼容调用  |
| OpenAI Responses        | `/v1/responses`                          | 标记支持 Responses 的 xAI 模型，以及工具生图流程 |
| Anthropic Messages      | `/v1/messages`                           | Claude 原生格式                                  |
| Google Gemini           | `/v1beta/models/{model}:generateContent` | Gemini 原生格式                                  |
| OpenAI Images           | `/v1/images/generations`                 | GPT Image、Imagen 等图像模型                     |

注意：

- API Key 在 `控制台 -> 令牌管理` 创建，通常以 `sk-` 开头。
- 令牌分组必须包含目标模型；模型存在不代表当前令牌一定有权限调用。
- 同一个模型可能支持多个协议，应以现网 `supported_endpoint_types` 为准。
- 不要把真实 API Key 写入代码仓库、截图、工单或群聊。

## 3. 设置公共变量

以下样例都使用两个环境变量：

```bash
export NAN_BASE_URL="https://cn.meta-api.vip"
export NAN_API_KEY="sk-替换为你的API密钥"
```

## 4. 查询当前可用模型

### 4.1 查询当前令牌实际可见的模型

这一步受令牌分组、模型限制和后台渠道配置影响，是调用前最可靠的检查：

```bash
curl -sS "$NAN_BASE_URL/v1/models" \
  -H "Authorization: Bearer $NAN_API_KEY" \
  | jq -r '.data[] | .id'
```

如果机器没有安装 `jq`，去掉最后一行即可查看原始 JSON。

### 4.2 查询公开模型、分组和支持协议

```bash
curl -sS "$NAN_BASE_URL/api/pricing" \
  | jq -r '.data[] | [
      .model_name,
      (.enable_groups | join(",")),
      (.supported_endpoint_types | join(","))
    ] | @tsv'
```

## 5. 当前模型快照

### 5.1 OpenAI

文本和代码模型：

- `gpt-5.3-codex-spark`
- `gpt-5.4`
- `gpt-5.4-mini`
- `gpt-5.5`
- `gpt-5.6-luna`
- `gpt-5.6-sol`
- `gpt-5.6-terra`

图像模型：

- `gpt-image-1.5`
- `gpt-image-2`

推荐分组：`vip`、`svip`、`土豪组`、`EMOXIA`。
协议：文本和代码模型使用 `openai`；图像模型使用 `image-generation`，部分也支持 `openai`。

### 5.2 Anthropic Claude

- `claude-fable-5`
- `claude-haiku-4-5-20251001`
- `claude-opus-4-6`
- `claude-opus-4-7`
- `claude-opus-4-8`
- `claude-sonnet-4-6`
- `claude-sonnet-5`

分组：`claude`、`土豪组`。
协议：`anthropic`、`openai`。

### 5.3 Google Gemini

- `gemini-2.5-pro-exp-03-25`
- `gemini-2.5-pro-preview-03-25`
- `gemini-3-pro-preview`
- `gemini-3.0-pro`
- `gemini-3.1-pro-preview`
- `gemini-3.5-flash`
- `gemini-omni-flash-preview`
- `[特价M]gemini-3.0-pro`
- `[特价M]gemini-3.1-pro`

分组：`gemini`、`土豪组`。
协议：`gemini`、`openai`。

带 `[特价M]` 的模型名必须完整传递。如果放在 Gemini 原生 URL 路径中，方括号和中文需要 URL 编码；更简单的方式是使用 OpenAI 兼容接口，把模型名放在 JSON 的 `model` 字段中。

### 5.4 Google Imagen

- `imagen-3.0-generate-002`

分组：`gemini`、`土豪组`。
协议：`image-generation`、`gemini`、`openai`。

### 5.5 xAI Grok

- `grok-4-1-fast-non-reasoning`
- `grok-4-1-fast-reasoning`
- `grok-4-20-non-reasoning`
- `grok-4-5-fast-reasoning`
- `grok-4.3`
- `grok-4.5`
- `grok-imagine-image`

协议：`openai`、`openai-response`。

### 5.6 DeepSeek

- `deepseek-v4-flash`
- `deepseek-v4-pro`

分组：当前公开在 `default`、`土豪组`。
协议：`openai`。

### 5.7 暂无公开模型的厂商

现网厂商元数据中还能看到 ChatGLM（智谱）和讯飞，但本次 `/api/pricing` 没有返回对应模型。因此本文不编造模型名；后台重新上架后，先用第 4 节命令取得准确模型名，再套用 OpenAI 兼容样例。

## 6. OpenAI / GPT 请求样例

### 6.1 GPT 文本模型

```bash
curl -sS "$NAN_BASE_URL/v1/chat/completions" \
  -H "Authorization: Bearer $NAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4",
    "messages": [
      {"role": "system", "content": "你是一个简洁、准确的中文助手。"},
      {"role": "user", "content": "用三句话解释什么是 API 网关。"}
    ],
    "max_completion_tokens": 256
  }'
```

只需替换 `model`，同一请求可用于 `gpt-5.4-mini`、`gpt-5.5`、`gpt-5.6-luna`、`gpt-5.6-sol`、`gpt-5.6-terra` 等当前可见模型。

### 6.2 Codex 模型

```bash
curl -sS "$NAN_BASE_URL/v1/chat/completions" \
  -H "Authorization: Bearer $NAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.3-codex-spark",
    "messages": [
      {"role": "user", "content": "写一个 Go 函数：输入整数切片，返回去重并升序排列后的结果。"}
    ],
    "max_completion_tokens": 512
  }'
```

### 6.3 GPT 图片模型

```bash
curl -sS "$NAN_BASE_URL/v1/images/generations" \
  -H "Authorization: Bearer $NAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2",
    "prompt": "一只在上海雨夜骑自行车的橘猫，电影感，霓虹灯，高细节",
    "size": "1024x1024",
    "quality": "high",
    "n": 1
  }'
```

完整的 Responses 生图、图片修改和本地技能包说明见 [AI 生图](./NAN_IMAGE_GENERATION.md)。

## 7. Claude 请求样例

### 7.1 Anthropic Messages 原生格式

```bash
curl -sS "$NAN_BASE_URL/v1/messages" \
  -H "x-api-key: $NAN_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-5",
    "system": "你是一个简洁、准确的中文助手。",
    "messages": [
      {"role": "user", "content": "比较 MySQL、PostgreSQL 和 SQLite 的适用场景。"}
    ],
    "max_tokens": 512
  }'
```

只需替换 `model`，可以调用本页列出的其他 Claude 模型。

### 7.2 Claude 的 OpenAI 兼容格式

```bash
curl -sS "$NAN_BASE_URL/v1/chat/completions" \
  -H "Authorization: Bearer $NAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-opus-4-8",
    "messages": [
      {"role": "user", "content": "审查这段需求，列出最容易遗漏的三个边界条件。"}
    ],
    "max_tokens": 512
  }'
```

## 8. Gemini 请求样例

### 8.1 Gemini 原生格式

```bash
curl -sS "$NAN_BASE_URL/v1beta/models/gemini-3.5-flash:generateContent" \
  -H "x-goog-api-key: $NAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          {"text": "给我一个五步上线检查清单。"}
        ]
      }
    ],
    "generationConfig": {
      "maxOutputTokens": 256,
      "temperature": 0.3
    }
  }'
```

不建议把 Key 放在 URL 查询参数中，因为 URL 更容易进入浏览器历史、代理日志和监控记录。

### 8.2 Gemini 的 OpenAI 兼容格式

```bash
curl -sS "$NAN_BASE_URL/v1/chat/completions" \
  -H "Authorization: Bearer $NAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-3.1-pro-preview",
    "messages": [
      {"role": "user", "content": "把下面标题改得更清晰：关于服务可靠性的一些思考。"}
    ],
    "max_tokens": 256
  }'
```

### 8.3 Imagen 图片生成

```bash
curl -sS "$NAN_BASE_URL/v1/images/generations" \
  -H "Authorization: Bearer $NAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "imagen-3.0-generate-002",
    "prompt": "极简东方茶室，清晨自然光，原木与白墙，建筑摄影",
    "size": "1024x1024",
    "n": 1
  }'
```

## 9. xAI Grok 请求样例

### 9.1 Chat Completions

```bash
curl -sS "$NAN_BASE_URL/v1/chat/completions" \
  -H "Authorization: Bearer $NAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-4-1-fast-reasoning",
    "messages": [
      {"role": "user", "content": "判断这个迁移方案的三个主要风险，并按严重程度排序。"}
    ],
    "max_completion_tokens": 512
  }'
```

只需替换 `model`，可以调用本页列出的其他 Grok 文本模型。

### 9.2 Responses

```bash
curl -sS "$NAN_BASE_URL/v1/responses" \
  -H "Authorization: Bearer $NAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-4.5",
    "instructions": "使用简体中文，先给结论。",
    "input": "解释零停机数据库迁移的基本思路。",
    "max_output_tokens": 512
  }'
```

### 9.3 Grok 图片模型

```bash
curl -sS "$NAN_BASE_URL/v1/responses" \
  -H "Authorization: Bearer $NAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-image",
    "input": "一座漂浮在云海上的未来图书馆，日出，宽银幕构图"
  }'
```

## 10. DeepSeek 请求样例

```bash
curl -sS "$NAN_BASE_URL/v1/chat/completions" \
  -H "Authorization: Bearer $NAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-pro",
    "messages": [
      {"role": "user", "content": "只回复 OK。"}
    ],
    "max_tokens": 256
  }'
```

推理模型可能先消耗一部分生成预算。如果 HTTP 200 但正文为空或内容被截断，先增大 `max_tokens`，不要直接判断模型不可用。

## 11. 流式请求样例

OpenAI 兼容协议加入 `"stream": true`，并让 `curl` 关闭输出缓冲：

```bash
curl -N "$NAN_BASE_URL/v1/chat/completions" \
  -H "Authorization: Bearer $NAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "messages": [
      {"role": "user", "content": "写一段 100 字左右的产品介绍。"}
    ],
    "stream": true,
    "stream_options": {"include_usage": true}
  }'
```

Claude 原生协议同样加入 `"stream": true`：

```bash
curl -N "$NAN_BASE_URL/v1/messages" \
  -H "x-api-key: $NAN_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-haiku-4-5-20251001",
    "messages": [
      {"role": "user", "content": "逐条列出上线前检查项。"}
    ],
    "max_tokens": 256,
    "stream": true
  }'
```

## 12. Python OpenAI SDK 通用样例

多数文本模型支持 OpenAI 兼容格式，因此一段代码可以切换 GPT、Claude、Gemini、xAI 和 DeepSeek：

```bash
python3 -m pip install openai
```

```python
import os

from openai import OpenAI

client = OpenAI(
    api_key=os.environ["NAN_API_KEY"],
    base_url="https://cn.meta-api.vip/v1",
)

response = client.chat.completions.create(
    model="deepseek-v4-flash",
    messages=[
        {"role": "user", "content": "用一句话说明请求是否成功。"},
    ],
    max_tokens=128,
)

print(response.choices[0].message.content)
```

## 13. 常见错误与排查

| 现象                   | 优先检查                                                                       |
| ---------------------- | ------------------------------------------------------------------------------ |
| `401` / `unauthorized` | API Key 是否完整、是否误带空格、令牌是否启用                                   |
| `model not found`      | 模型名是否完全一致，尤其是日期后缀、点号、连字符和 `[特价M]` 前缀              |
| `No available channel` | 令牌分组是否包含模型、后台渠道是否在线、模型是否支持当前协议                   |
| Claude 请求失败        | 是否使用 `/v1/messages`、`x-api-key`、`anthropic-version` 和 `max_tokens`      |
| Gemini 请求失败        | URL 是否为 `/v1beta/models/{model}:generateContent`，是否使用 `x-goog-api-key` |
| DeepSeek 返回空正文    | 增大 `max_tokens`；推理过程可能先消耗生成预算                                  |
| 流式响应看起来卡住     | 使用 `curl -N`，记录发生时间、模型名和响应头中的请求 ID                        |
| 图片响应没有 URL       | 检查 `data[0]` 是否返回 `b64_json`；输出形态取决于模型和渠道                   |

排障时建议保留响应头：

```bash
curl -i "$NAN_BASE_URL/v1/chat/completions" \
  -H "Authorization: Bearer $NAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "messages": [{"role": "user", "content": "ping"}],
    "max_completion_tokens": 64
  }'
```

提交问题时提供发生时间、模型名、请求协议、令牌名称、完整错误文本、HTTP 状态码和响应头中的请求 ID；不要提供完整 API Key。
