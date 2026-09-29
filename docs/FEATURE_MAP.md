# 功能地图

核对日期：2026-09-29。用于定位代码，不替代代码和现场验证；覆盖已核对的渠道运营与协议转换能力，其他模块按任务逐步补充。更改相关功能时同步本表，不把历史方案写成当前能力。

| 能力/入口 | 主要实现 | 接口与数据 | 边界 |
|---|---|---|---|
| 渠道管理：列表、已用/剩余、优先级、权重 | [渠道表格](../web/src/components/table/channels/index.jsx)、[列定义](../web/src/components/table/channels/ChannelsColumnDefs.jsx)、[渠道模型](../model/channel.go) | `/api/channel/`；主库 channels，余额按渠道支持情况查询 | used_quota 是累计额度，不是时间段消耗；余额不等于日志统计 |
| 渠道管理：时段用量视图 | [用量工具栏](../web/src/components/table/channels/ChannelUsageToolbar.jsx)、[组合列表接口](../controller/channel_usage.go)、[用量模型](../model/channel_usage.go) | 管理员 `/api/channel/usage`；主库渠道过滤 + 共享日志聚合定义，精确半开区间 | 最长 30d、全量消耗排序后分页、完整标签汇总、全筛选范围占比；消费/错误日志，不代表上游成本；已删除渠道单列 |
| 数据看板：分组统计与渠道用量入口 | [统计面板](../web/src/components/dashboard/ChannelMonitorPanel.jsx)、[controller](../controller/monitor.go)、[model](../model/monitor.go) | 管理员 `/api/monitor/channels`；日志库聚合后主库补渠道信息 | 默认 24h、最多 30d、默认 60s 缓存；看板通过 exact_range 使用精确半开区间，跳转携带时段；旧调用保留对齐边界；consume/error 日志口径 |
| 使用日志：厂商/渠道筛选与分组汇总 | [使用日志界面](../web/src/components/table/usage-logs)、[范围解析](../model/log_scope.go)、[现有说明](LOG_GROUP_CHANNEL_FILTERS.md) | `/api/log/`、`/api/log/stat`、`/api/log/group-summary`；日志及渠道范围 | 汇总消费记录口径与监控 consume/error 口径有差异；不能直接混用成功率/请求数 |
| 渠道周期额度策略 | [controller](../controller/channel_period_quota.go)、[service](../service/channel_period_quota.go) | `/api/channel/:id/quota_usage`、`/api/channel/tag/quota_usage`；策略周期用量 | 面向配额控制与策略周期，不替代任意时间范围统计 |
| 渠道路由与权重 | [缓存路由](../model/channel_cache.go)、[能力选择](../model/ability.go) | 分组/模型/优先级筛选后的渠道选择 | 权重影响请求分配，不保证金额占比；不能以累计花费直接反推合理权重 |
| 工具选择与历史思考转换 | [请求转换](../service/convert.go)、[工具选择](../service/claude_tool_choice.go)、[Chat/Responses](../service/openaicompat/chat_to_responses.go) | `/v1/messages` 转 Chat；Chat 消息级 tools 由请求 DTO 保留 | 明确 any/指定工具/并行要求，不强迫 auto 调工具；不支持的动态工具跨协议转换明确拒绝；不保证上游执行 |
| 视频请求与输出预算 | [请求 DTO](../dto/openai_request.go)、[K3 输出别名](../dto/kimi_output_limit.go)、[Claude 转换](../relay/channel/claude/relay-claude.go) | 复用 relay 鉴权/渠道选择；视频字符串与对象保留，max_completion_tokens 使用同一有效上限 | 不支持的视频转 Claude 返回错误；不提取视频帧或保证上游理解；不截断回答；不改变价格与实际 usage |
| Chat 转 Messages 流式交付 | [流处理](../relay/channel/openai/relay-openai.go)、[工具块状态](../service/chat_to_claude_stream.go)、[scanner](../relay/helper/stream_scanner.go) | 内容立即发送，结束元数据单独保留以接收 usage；错误交给 controller 退款 | 不以心跳计首字；已输出不重试；不完整流无成功终态；不代表上游首字延迟 SLA |

权限最终以 [API 路由](../router/api-router.go) 为准。新增渠道分析先对比统计面板、渠道管理和使用日志三处，确认数据定义、权限、分页/排序与跳转条件一致后再选方案。
