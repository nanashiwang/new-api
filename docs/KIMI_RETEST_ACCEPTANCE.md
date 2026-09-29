# K3 网关整改与复测验收

## 范围

本清单区分四项整改与一项验收流程：已知假成功与计费、动态工具消息、视频格式、完整请求体大小，以及三协议与页面验收。参数语义、供应商执行工具、视频理解、输出限制与首字长尾不自动归因于网关；不能把字段保真测试写成模型能力测试。

测试必须记录实际入口、渠道、模型、版本、请求 ID 和时间。供应商侧报告可以提供反例，但相同模型名称或“上游”称谓不证明经过了同一条链路。报告的总量、含思考首字、可见正文首字和失败样本需要统一口径后比较。

## 验收矩阵

| 项目 | 元衡需证明 | 本地可做的验证 | 不能据此宣称 |
|---|---|---|---|
| 已知假成功 | 特征匹配后明确失败，真实预扣被退还，不产生正常消费记录，不因文字里出现“unavailable”误拒正常回答 | 合并伙伴实现后，用其支持的准确特征覆盖 JSON/SSE、拆块、正常引用反例及三种资金来源 | 普通断流退款测试不能替代“成功正文是失败提示”的识别；不能据 usage 字段直接调整历史余额 |
| 动态工具 | 声明、工具选择、参数完整，未提供 content 不被网关填成 null；显式 null/空字符串不偷偷删除 | DTO 深拷贝及最终出站体对比；区分调用者原始字段与网关生成字段 | 上游一定会调用工具，或所有 KVV 动态工具用例已通过 |
| 视频 | video_url 字符串/对象保持原结构，对象内扩展字段保留，不支持的转换明确失败 | 数据结构、出站 payload 对比；使用合成数据而非客户视频 | HTTP 200 代表模型读到了视频，或伪视频夹具证明理解能力 |
| 请求体 | 管理配置统一覆盖 Chat、Messages、Responses；超限413、解压后限制、缓存清理 | 192 MiB存储与本地HTTP字节转发 SHA-256 一致；精确边界、伪造/未知长度、gzip膨胀测试 | 192 MiB 是原视频大小，或供应商/CDN/Nginx也接受相同体积 |
| 验证 | 三协议正确返回完整 SSE，失败计费独立验收；站点与KVV按实际入口核查 | 本机HTTP模拟上游、隔离数据库和主前端构建 | 模拟上游等于真实供应商；编译通过等于页面和完整KVV验收通过 |

## 192 MiB 部署检查

- 使用已有管理员配置 `ResponsesRequestBodyLimitMB=192`，同时确认 `MAX_REQUEST_BODY_MB` 不低于192。环境变量 `RESPONSES_REQUEST_BODY_LIMIT_MB` 只是初始化来源，数据库已保存的管理配置仍需核对。
- 不通过代码迁移强改管理员既有值。本地保留默认值和更小的全局硬限制；Messages 接入同一业务上限后，部署前必须核对旧配置是否仍为20。
- 检查入口反代与 CDN 的上限。应用接受192 MiB不能抬高外层较低限制；不能把上传超时直接认定成413。
- 限制针对解压后的完整请求体。128 MiB二进制视频转Base64约170.67 MiB，还要计入其他消息、工具和JSON开销。
- 本地192 MiB测试验证缓存和HTTP字节转发，不经过完整模型语义转换，也不是真实供应商的大视频验收。真实端到端验证须单独授权、固定渠道、串行小样本，记录资源占用与上游实际收到的完整字节。

## 本地检查命令

```sh
go test ./dto ./common ./middleware ./controller
NEW_API_LARGE_BODY_TEST=1 go test ./common -run '^TestRequestBody192MiBStorageIntegrity$' -count=1
NEW_API_LARGE_BODY_TEST=1 go test ./middleware -run '^TestDialog192MiBHTTPByteIntegrity$' -count=1
go test ./controller -run '^TestKimiRetestThreeProtocolsOverLocalHTTP$' -count=1
go test ./controller -run '^(TestRelayChatAndMessagesInterruptedOutputRefunds|TestRelayResponsesInterruptedOutputRefundsAllFundingSources)$' -count=1
make verify-notes
```

`TestKimiRetestThreeProtocolsOverLocalHTTP` 使用真实本机HTTP连接但仅模拟供应商，不使用客户密钥。完整KVV是单独的用例集合；必须记录工具版本、原始结果和跳过项，不从仓库中不存在的独立页面推定通过。

## 待补证据

- 已知“假成功”识别的伙伴提交及对应部署版本。当前仅找到普通协议失败/断流退款实现，不能据此把第一项勾成通过。
- 客户相同请求的脱敏原始请求/响应和实际入口、渠道映射；不保存API密钥、真实客户视频或聊天正文到公开仓库。
- 真实供应商下的工具调用结果和视频内容问答；上线后的单渠道、同时间/上下文口径延迟对照。
- 首页、文档及独立KVV入口的实际浏览器验收。当前仓库确认有首页和 `/docs` 路由，未找到独立KVV页面，需提供其地址或实现位置。
- 七条历史异常计费候选的原始响应与账本证据；在证据补齐前不改客户余额。

## 本轮本地结果

- Go全仓功能测试、主前端构建及改动文件格式检查通过。
- 参数保真、三协议HTTP与请求体边界的定向竞态测试通过。
- 首页和`/docs`使用模拟只读API进行本地浏览器冒烟，渲染正常且无JavaScript异常；不代表线上验收。
- 既有Responses取消退款用例暴露的日志计数器竞态已通过局部同步修复；包含该用例的日志、请求体和三协议退款定向竞态检查连续三轮通过。未执行全仓竞态或生产负载测试。
