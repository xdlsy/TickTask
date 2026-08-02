# ADR-0004: OpenAI-Compatible AI Integration

**Status**: Accepted
**Date**: 2026-06-07
**Context**: [ai](../modules/ai.md) | [AI Schedule Generation](../flows/ai-schedule-generation.md) | [Schedule Revision](../flows/schedule-revision.md)

## Context

TickTask 需要 AI 能力来实现任务自动分类、日程生成和优先级排序。需要选择 LLM 集成方案。

## Decision

使用 **OpenAI 兼容 API** 作为 LLM 集成接口。通过配置 `baseURL` 和 `apiKey` 支持任意 OpenAI 兼容的 LLM 提供商（OpenAI、Azure OpenAI、本地 Ollama 等）。使用标准 `net/http` 直接调用，不使用 SDK。

## Alternatives Considered

| 方案 | 优点 | 缺点 |
|------|------|------|
| **OpenAI 兼容 API + net/http** (选中) | 供应商无关、零 SDK 依赖、灵活切换 | 需手动处理 HTTP 细节 |
| OpenAI Go SDK | 类型安全、官方维护 | 锁定 OpenAI 供应商、额外依赖 |
| LangChain Go | 抽象层丰富、多模型支持 | 过度抽象、依赖链长、学习成本高 |
| 本地模型 (Ollama) | 无 API 费用、数据不出本机 | 需要本地 GPU、模型质量不稳定 |

## POS (正面影响)

- **供应商无关**：配置 baseURL 即可切换 LLM 提供商
- **零 SDK 依赖**：标准 `net/http` 实现，无第三方 SDK
- **优雅降级**：未配置 API Key 时 AI 功能静默降级（AIService 返回 nil）
- **接口隔离**：`LLMClient` 接口支持测试 mock 替换
- **灵活配置**：环境变量 `TT_AI_API_KEY` 可覆盖配置文件

## NEG (负面影响)

- **手动 HTTP 处理**：需自行处理超时、错误码、JSON 解析
- **Markdown 响应**：LLM 有时返回 markdown code fence 包裹的 JSON，需 `extractJSON()` 剥离
- **长延迟**：LLM 响应可能耗时数十秒，需 360s 前端超时
- **无流式响应**：当前实现等待完整响应，未使用 SSE streaming

## IMP (实施要点)

- 配置存储在 `configs/config.yaml` 的 `ai` 段，API Key 明文存储（不入 Git）
- 构造函数 `NewOpenAIClient(apiKey, baseURL, model)` 默认 GPT-4o-mini
- 中文 Prompt 模板明确要求 "只返回 JSON"
- Handler 层使用 `context.WithTimeout` (30s/60s) 防止无限等待
- 前端长请求（日程生成/修订）使用 360s 超时

## REF (参考)

- [ai 蓝图](../modules/ai.md)
- [service 蓝图](../modules/service.md)
- [AI Schedule Generation 流程](../flows/ai-schedule-generation.md)
- [Schedule Revision 流程](../flows/schedule-revision.md)
