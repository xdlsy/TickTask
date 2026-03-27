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

	// PriorityPrompt 优先级建议 Prompt
	PriorityPrompt = `你是一个时间管理专家。请根据以下待办任务，为用户推荐处理优先级。

任务列表:
%s

请以 JSON 数组格式返回推荐的任务 ID 顺序（按优先级从高到低）：
{
    "priority_order": ["task_id_1", "task_id_2", ...]
}`
)
