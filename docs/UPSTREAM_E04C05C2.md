# e04c05c2 选择性迁移（compat.16）

来源：QuantumNous/new-api `e04c05c2b929c0df6cc35227f860cf6e19d480c7`。
起点：本地 main `f05de46e6`；保持 main，不切换分支。

## 冲突评估与适配

上游修改新版 TypeScript/Input/Combobox，本地是 React/Semi UI，没有对应组件，不能直接 cherry-pick。按行为手工迁移：

- 分组倍率、特殊分组倍率、工具价格的 Semi InputNumber 使用 `step=0.0001`、`precision=4`；超过四位按组件规则舍入，保留非负下限。
- 充值分组倍率在本地仍是 JSON 编辑，本来支持四位小数，不迁移上游充值表格。
- 本地没有上游日志隐私状态，新增浏览器持久化的「分组隐私模式」，遮挡分组/厂商筛选框及其 portal 下拉浮层。默认关闭，仅用于筛选区域的视觉遮挡，不是数据脱敏或权限控制。
- 保留厂商筛选、历史分组输入、渠道关键词筛选、分组汇总、缓存统计、工具模型前缀覆盖以及现有 JSON 配置 API。
- 后端生产代码和数据库结构不变；新增配置精度与工具价格查询回归。

## 验证

- `go test ./...`、`go build` 通过。
- `go test -race ./setting/ratio_setting ./setting/operation_setting` 通过。
- 前端 `node --test scripts/*.test.mjs`：20 项通过。
- `bun run build`：图片 Playground 与主站构建通过；本机没有 `/usr/bin/chromium`，预渲染脚本按现有规则跳过，后端 meta 渲染兜底。
- 使用真实本地页面和模拟 API：分组倍率及工具价格逐字输入 0.0001、保存请求、刷新读取；工具价格 0.04、零值、负数下限与五位小数舍入。
- 浏览器验证分组/厂商 portal 选项计算样式为 disc，隐私开关刷新持久化，以及原始分组值仍用于查询；390px 暗色布局和关闭隐私后恢复显示通过。

## 发布说明

v1.0.0-compat.16：支持四位小数分组倍率与工具价格；补齐日志分组筛选隐私遮挡。继续使用已有 GitHub Actions 构建四个平台附件和 GHCR AMD64/ARM64 镜像。发布结果以 Actions、Release 和版本 manifest 终态为准；本轮不部署生产。
