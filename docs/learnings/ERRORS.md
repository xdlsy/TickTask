# 错误日志

命令失败和集成错误。

---

### [ERR-20260607-001] 前端 TypeError: Cannot read properties of null (reading 'map')

- **触发**: 点击"确定"应用日程修订，或 AI 生成日程后
- **表现**: 前端崩溃，显示"应用修订失败，请重试"
- **根因**: Go `var x []T`（nil slice）序列化为 JSON `null`；前端 `.map()` 对 null 抛出 TypeError
- **修复**: 后端用 `make([]T, 0)` 初始化；前端加 `?? []` 空值守护
- **涉及文件**: `schedule_service.go:492,758` / `schedule.ts:144,220`
- **commit**: `9eea638`
- **模式**: Go nil slice → JSON null → TS TypeError
- **防范**: 所有返回数组的 Go 函数使用 `make([]T, 0)`，前端在数组操作前做空值检查

---

### [ERR-20260607-002] AI 功能调用返回 JSON 解析失败

- **触发**: AI 任务分类、日程生成、优先级排序
- **表现**: `json.Unmarshal` 报错 "invalid character '˜' looking for beginning of value"
- **根因**: LLM 返回内容被 markdown 代码围栏包裹（\`\`\`json ... \`\`\`）
- **修复**: 使用 `extractJSON()` 辅助函数剥离代码围栏
- **涉及文件**: `service/ai_service.go` (`extractJSON`)
- **模式**: LLM 输出不可控 → 需防御性解析
- **防范**: 所有 AI 响应经过 `extractJSON()` 预处理

<!-- HUMAN_REVIEW -->
