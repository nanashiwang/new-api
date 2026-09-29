# Agent Note: 官方 KVV 基线与本地转发测试对齐

Status: implemented

## Problem

本地模拟上游把任何包含 content 的动态工具消息都判成400，但官方 KVV 的成功用例使用 system + content="" + tools，并要求返回真实结构化工具调用。由错误模拟器得到的绿灯不能说明与官方兼容。门户 quick 摘要的跳过项和部分覆盖也不能等价于完整官方验收。

## Existing capabilities and impact

- 官方依据为 MoonshotAI/Kimi-Vendor-Verifier 提交 `66092cf444c97356c0e11c5078c67116390615d9`，本地核对 README、params、k3_features 的动态工具/格式/工具选择/思考断言及流式包装。
- 现有 DTO 已原样保留显式空字符串 content；问题首先位于 `controller/relay_kimi_compat_test.go` 的模拟器及注释，不据此推断生产会拒绝空字符串。
- 复用现有控制器、DTO 和流式解析测试，不增加 KVV 门户，不修改生产渠道、计费或参数策略。
- [复测范围纠偏](../bug-fix/2026-09-29-kimi-retest-scope-correction.md)与[请求流式完整性](../bug-fix/2026-09-29-kimi-request-stream-integrity.md)的保真原则保留；官方依据、用例预期与报告映射由本笔记补充。日志同步与渠道用量笔记无重叠。

## Decision

- 固定官方提交，记录 suite/nodeid、stream/nostream、入口、模型、渠道、重试及原始跳过原因；官方仓库不改用例和断言。
- 本地以官方明确的 system 空字符串正例、user/assistant 角色和非空 content 反例验证转发与错误传递；省略和 null 单独做保真检查，不编造官方对 null 的结论。
- 修正本地模拟器，成功时返回 tool_calls，验证首条、后续和末尾动态工具消息的流式/非流式响应。该验证不冒充供应商实际执行。
- 流式模拟器声明自己支持 stream_options，复用既有渠道能力开关；不修改未知兼容主机的保守默认值，也不修改生产渠道配置。
- 关闭思考用例按官方原文记 skip，不能推导为模型能力豁免；low/high/max 与启用的实际断言分开记录，已跳过的长度单调测试不作为硬门槛。
- 门户 quick 摘要映射为“待原始JSON核对的部分覆盖”；S1/S2、多轮/图像/视频额外验收单列，不能说属于官方quick全覆盖。

## Alternatives considered

- **继续扩充本地模拟器来代替官方测试**：无费用且确定性高，但会把自己写错的假设测成通过。模拟器只覆盖网关保真，供应商验收使用固定的官方原始套件。
- **立即在线跑完整KVV**：证据最直接，但入口与专用密钥、费用及限额尚未确认，官方重试和长输出会扩大调用量。本轮只做离线对齐，不使用历史密钥或发起压测。

## Consequences

上游基线将来可能变化，需要重新核对提交与断言，不能跟随main漂移。官方源码自身也有暂跳、随机性重试和注释/断言差异；保留原始证据，不擅自修测试。尚未取得门户原始JSON，不能核验其16项的具体nodeid、参数和HTTP200内是否存在错误。

本地回归具有明确来源，避免空字符串正例被模拟器误拒绝。代价是模型执行、性能与完整KVV仍需要额外的在线验证，不能用便宜的模拟用例替代。运行时仅修正DTO注释，没有修改请求序列化行为、价格、权限或路由。

## Verification

- 修复模拟器前，官方空字符串形状在OpenAI/Moonshot两种渠道的定向用例中复现错误400。
- 修复后7个场景×2种渠道×2种流式模式共28个本地转发子用例通过；断言入站/出站JSON一致，正例返回模拟结构化工具及完整参数/终态，明确反例保持400。
- controller/dto相关用例在 `go test -race -p 2 -timeout 60s ... -count=1` 下通过。
- `go test -p 2 -timeout 120s ./...` 全仓通过。使用Go 1.25.1、GOMAXPROCS=2、GOPROXY=off。
- 只读取官方源码，没有执行官方在线套件，没有使用客户API密钥或改变服务器。完整KVV、门户quick原始JSON、S1/S2及图像视频能力仍未验收。
