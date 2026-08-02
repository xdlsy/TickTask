#!/bin/bash
# 学习改进激活器 Hook
# 在 UserPromptSubmit 时触发，提醒 Claude 关于学习捕获
# 保持输出最少（约 50-100 tokens）以最小化开销

set -e

# 将提醒输出为系统上下文
cat << 'EOF'
<self-improvement-reminder>
完成此任务后，评估是否出现了可提取的知识：
- 通过调查发现了非显而易见的解决方案？
- 为意外行为找到了变通方案？
- 学到了项目特定的模式？
- 错误需要通过调试才能解决？

如果有：按照学习改进技能的格式记录到 docs/learnings/。
如果价值高（重复出现、广泛适用）：考虑技能提取。
</self-improvement-reminder>
EOF
