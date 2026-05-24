package ai

const (
	// ClassifyPrompt 任务分类 Prompt
	ClassifyPrompt = `你是一个时间管理专家。请根据以下任务信息判断其重要性和紧急程度。

任务标题: %s
任务描述: %s
截止时间: %s

请仅返回 JSON 格式，不要包含任何其他内容：
{
    "important": true/false,
    "urgent": true/false,
    "reason": "判断理由（简短）"
}`

	// SchedulePrompt 日程生成 Prompt
	SchedulePrompt = `你是一个时间管理专家。请根据以下信息为用户安排今日日程。

可用时间: %s 到 %s
番茄时长: %d 分钟
休息时长: %d 分钟
长休息间隔: 每 %d 个番茄后

待办任务:
%s

请以 JSON 格式返回日程安排，包含任务时间段：
{
    "schedule": [
        {
            "task_id": "任务ID",
            "title": "任务标题",
            "start_time": "HH:MM",
            "end_time": "HH:MM",
            "pomodoro_count": 番茄数量
        }
    ]
}`

	// ClassifyByTextPrompt 根据文本分类任务 Prompt
	ClassifyByTextPrompt = `你是一个时间管理专家。请根据以下任务信息判断其重要性和紧急程度，并推荐合适的象限分类。

任务标题: %s
任务描述: %s

请仅返回 JSON 格式，不要包含任何其他内容：
{
    "important": true/false,
    "urgent": true/false,
    "quadrant": 1-4,
    "reason": "判断理由（简短）",
    "suggested_tags": ["标签1", "标签2"]
}`

	// ReschedulePrompt 被打断后重新排程 Prompt
	ReschedulePrompt = `你是一个时间管理专家。用户正在执行番茄钟时被打断了，请根据当前情况调整今日剩余日程。

被打断的任务: %s
被打断时已完成: %d 分钟 / 总共计划: %d 分钟
打断原因: %s
当前时间: %s
今日剩余可用时间: %s 到 %s

今日剩余待办任务:
%s

请以 JSON 格式返回调整后的日程建议：
{
    "adjusted_schedule": [
        {
            "task_id": "任务ID",
            "title": "任务标题",
            "start_time": "HH:MM",
            "end_time": "HH:MM",
            "adjustment": "unchanged/postponed/shortened",
            "reason": "调整理由（简短）"
        }
    ],
    "summary": "整体调整说明"
}`

	// DailyInsightsPrompt 每日洞察 Prompt
	DailyInsightsPrompt = `你是一个时间管理专家和生产力教练。请根据用户今天的工作数据，给出简短有力的洞察和建议。

今日日期: %s
完成番茄数: %d
总专注时间: %d 分钟
完成任务数: %d
被打断次数: %d
任务分布: %s

请以 JSON 格式返回：
{
    "productivity_score": 1-100,
    "peak_hours": "高产时段描述",
    "achievements": ["亮点1", "亮点2"],
    "suggestions": ["建议1", "建议2"],
    "motivation": "一句鼓励的话"
}`

	// PriorityPrompt 优先级建议 Prompt
	PriorityPrompt = `你是一个时间管理专家。请根据以下待办任务，为用户推荐处理优先级。

任务列表:
%s

请以 JSON 数组格式返回推荐的任务 ID 顺序（按优先级从高到低）：
{
    "priority_order": ["task_id_1", "task_id_2", ...]
}`
)
