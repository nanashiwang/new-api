# 官方 KVV 对齐基线

核对日期：2026-09-29。基线来自 **MoonshotAI/Kimi-Vendor-Verifier**，提交固定为：

```text
66092cf444c97356c0e11c5078c67116390615d9
```

这是一份验收依据和结果记录规则，不是“元衡或某条渠道已经通过官方KVV”的报告。官方套件是独立测试代码，不以本站存在一个KVV网页为前提；第三方 kvv-portal 的 quick profile 需另外核对版本和用例映射。这里核对的契约套件针对原生Chat Completions，不能替代本站Messages/Responses的独立兼容验收。

## 直接核对的官方依据

- [官方说明](https://github.com/MoonshotAI/Kimi-Vendor-Verifier/blob/66092cf444c97356c0e11c5078c67116390615d9/README_zh.md)
- [动态工具正反例](https://github.com/MoonshotAI/Kimi-Vendor-Verifier/blob/66092cf444c97356c0e11c5078c67116390615d9/tests/k3_features/test_dynamic_tools.py)
- [格式约束](https://github.com/MoonshotAI/Kimi-Vendor-Verifier/blob/66092cf444c97356c0e11c5078c67116390615d9/tests/k3_features/test_response_format.py)
- [工具选择](https://github.com/MoonshotAI/Kimi-Vendor-Verifier/blob/66092cf444c97356c0e11c5078c67116390615d9/tests/k3_features/test_tool_choice.py)
- [思考档位和跳过原因](https://github.com/MoonshotAI/Kimi-Vendor-Verifier/blob/66092cf444c97356c0e11c5078c67116390615d9/tests/k3_features/test_thinking_effort.py)
- [流式/非流式展开方式](https://github.com/MoonshotAI/Kimi-Vendor-Verifier/blob/66092cf444c97356c0e11c5078c67116390615d9/tests/k3_features/conftest.py)
- [参数验证](https://github.com/MoonshotAI/Kimi-Vendor-Verifier/blob/66092cf444c97356c0e11c5078c67116390615d9/tests/params/test_params.py)
- [入口、模型与密钥配置](https://github.com/MoonshotAI/Kimi-Vendor-Verifier/blob/66092cf444c97356c0e11c5078c67116390615d9/tests/conftest.py)

## 关键断言

| 对象 | 此基线的实际要求 | 本地处理 |
|---|---|---|
| system 动态工具 | `content: ""` + `tools` 是成功用例；required 应得到结构化 tool_calls 和结束原因 tool_calls | 空字符串原样保留；模拟器接受该正例，检查最终出站体和模拟工具响应完整性 |
| 动态工具位置 | 首条、后续、末条 system 均有成功用例 | 对这些位置覆盖 OpenAI兼容/Moonshot 两种渠道及流式/非流式 |
| 动态工具反例 | user/assistant 角色，以及 system 非空 content 与 tools 同时出现，应400 | 不偷偷删除非法字段；模拟上游400保持错误语义 |
| 省略/null | 不从空字符串正例推导全部空值等价 | 保留显式 null；省略仍省略。它们的本地保真检查不冒充官方供应商能力结论 |
| 工具选择 | auto、none、required 有不同断言；指定函数形式有官方skip | 不强制所有请求 required，不将skip当通过 |
| 思考档位 | 源码列出 low/high/max | 不写成只有low/high；不以单次长度递增证明档位正确 |
| 关闭思考 | 两个函数标记skip，原因为断言不正确 | 记原始skip，不据此声称“原厂恒思考，所以豁免”；需要独立规格依据和验收约定 |
| 推理长度 | 单调增长和默认档接近max的用例因噪声被跳过，其他部分带rerun | 不把单次68→141→343等结果升级成硬门槛，记录重跑和不稳定结果 |
| 流式 | k3_features中的client/hclient用例通过fixture展开stream与nostream | 两种模式分别报告，不只统计HTTP200 |

官方 `test_thinking_effort.py` 的文件说明与部分实际优先级断言存在不一致时，以原始运行结果和具体nodeid呈现，并向维护者确认；不自行修改官方测试或静默改写线上参数来消除差异。

## 客户 quick 摘要如何映射

目前只有摘要，未取得原始JSON、kvv-portal代码版本及选例规则，下表只能作候选映射：

| 摘要模块 | 候选官方用例 | 结论边界 |
|---|---|---|
| no_param 两项 | `tests/params/test_params.py::test_no_param_succeeds` | 不等于完整参数合法值和非法值矩阵 |
| text/json_object/json_schema_strict 六项 | `test_text_default`、`test_json_object`、`test_json_schema_strict` 各两种流式模式 | 不等于所有格式反例、schema组合或工具schema验证 |
| required/none 四项 | `test_tool_choice_required_forces_call`、`test_tool_choice_none_forbids_call` 各两种模式 | 不覆盖auto、动态工具、重复工具名或工具结果回环 |
| thinking 开档/关档 | 需原始nodeid确认具体函数与流式展开 | 不根据模块名猜两个skip的函数，更不能把skip计为pass |

“14通过、0失败、2跳过”仅能描述该次已选样例。未提供的用例按未测记录，不能把14/14换成全功能100%。测试入口、实例、渠道或版本不同，结论不得继承；相同模型名称不足以证明链路相同。

## 官方验收与补充验收分开

1. **官方契约套件**：params、k3_features、prompt_tokens、tool_call_json_schema，各保留选例范围、完整nodeid、JUnit原始结果及skip/rerun原因。
2. **官方能力基准**：README列出的OCRBench、MMMU Pro Vision、BEAM等按各自说明执行；不把门户的简化红图或短上下文测试冒称为这些完整基准。
3. **业务补充**：S1/S2时延/并发门槛、短多轮回环、图像/视频、192 MiB及假成功计费独立记录，不套用官方契约套件的通过数量。

源码中 `tests/params/test_wrong_param_rejected` 使用“请求未成功”的断言，某些5xx/网络错误也可能落入该条件。保留官方原始PASS，同时在独立诊断中标注HTTP状态和原因；不能把这种PASS解释为正确的400参数拒绝，更不能作为渠道健康证据。

## 可复现执行规则

- 使用固定提交，不在同一报告里随main漂移；检查官方源码未修改。
- 明确 `KIMI_BASE_URL`、`MODEL_NAME`、thinking格式、stream能力配置、渠道和版本。不得因变量未设置而退回官方默认入口，更不能静默换到备用渠道。
- 密钥通过专用测试环境提供，不放命令行参数、仓库、截图或报告。固定单渠道只影响经授权的测试范围，不改变普通用户路由。
- 先收集用例清单，再按授权的费用/请求预算运行；部分用例写死较大的输出上限并带重试，不能只按“测试函数数量”估计请求次数。预算中止必须记缺失项。
- 不修改官方断言、不把失败改skip、不隐藏首次失败和重试，不改token报数或返回正文来通过测试。
- 保留官方原始结果和独立诊断。HTTP200、SSE完整性、工具参数可解析、计费状态和模型语义不是同一层结论。

经授权后，在固定版本的官方检出目录执行各套件，示例不代表本轮已运行：

```sh
# 首先确认入口和模型已明确配置；密钥由专用环境提供。
: "${KIMI_BASE_URL:?必须明确测试入口}"
: "${MODEL_NAME:?必须明确模型}"
uv run pytest tests/params tests/k3_features tests/prompt_tokens \
  --base-url "$KIMI_BASE_URL" --smoke-model "$MODEL_NAME" \
  --think-mode kimi --collect-only -q

# 费用和测试范围批准后，分别运行并保留结果；不要把密钥写入参数。
uv run pytest tests/k3_features \
  --base-url "$KIMI_BASE_URL" --smoke-model "$MODEL_NAME" \
  --junitxml=k3-features.xml -ra -v
```

完整套件是否通过、供应商是否满足SLA及视频理解能力，必须以实际输出证明。本地模拟器、源码阅读和用例对齐都不是这些在线结果。

缓存/思考字段的补充诊断见[用量字段验收](KIMI_USAGE_FIELDS.md)：字段存在不等于缓存命中，缺字段不等于没有思考；此诊断独立于官方原始用例，不据此修改官方断言。
