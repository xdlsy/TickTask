# TickTask - Key Decisions Log

## Phase 0
- **2026-05-24**: 确定 Brownfield 路线 (Phase 8 产品进化)
- **2026-05-24**: 简报级别选择 Complete (完整)
- **2026-05-24**: 设计文件存放于 docs/

## Phase 1

### Step 01a: Client Profile
- **Organisation**: 内部产品团队，~30人，软件开发行业
- **Key People**: 用户身兼发起人/决策者/技术负责人三角色
- **Decision Culture**: 个人快速决策，无审批链条
- **Internal Driver**: 工作杂乱 → 遗忘/决策瘫痪 → 加班 → 需要自动化排程和效率分析
- **Working Style**: 直接沟通，想清楚再动手

### Step 02: Vision
- **Vision Statement**: 打造智能效率助手，用户告诉它"做什么"，它来规划"怎么做"——番茄钟粒度日程 + 打断自动重排 + 效率分析
- **Core differentiator**: 不只是列出任务，而是安排好什么时间做什么；打断后自动调整而非被动提醒
- **Target experience**: 轻松，无需费心思规划

### Step 03: Positioning
- **Positioning**: 面向软件开发团队的智能效率助手，像秘书一样自动排程 + 灵活调整 + 延期提醒建议
- **3 Differentiators**: (1) 番茄钟粒度自动排程 (2) 冲突灵活调整 (3) 延迟主动提醒+建议
- **Category**: 智能效率助手（非传统 To-Do）
- **First impression goal**: "像个秘书一样"

### Step 05: Business Model
- **Model**: 无商业模式 — 开源项目，非盈利，内部工具
- **Rationale**: 解决自己和团队的实际效率问题，不计划商业化

### Step 07: Target Users
- **Primary**: 软件开发团队成员 — 脑记排优先级，面临打断/琐事优先/决策瘫痪/规划成本高四大痛点
- **Secondary**: 团队 Leader — 阶段二才考虑，需查看团队负载和效率
- **Usage pattern**: 录入任务 → 自动排程 → 跟着执行（风格无关，统一入口）
- **Core failures**: 打断忘记、琐事占位、不知道先做什么、自己规划费时费力

### Step 07a: Product Concept
- **Core structure**: AI驱动的自动排程引擎 —— 录入 → AI排程 → 执行 → 调整 → 分析 → 反馈
- **Key decision**: AI是核心决策者，规则算法仅作降级方案
- **Unique loop**: 越用越懂你 —— 历史数据反哺下一次排程
- **7-step user journey**: 录入 → 排程 → 执行 → 调整 → 延迟建议 → 收工总结 → 数据反哺

### Step 08: Success Criteria
- **Primary**: 下班时间从凌晨12:00 → 晚上8:30（减3.5h加班）
- **Secondary**: 日任务完成率 ≥ 70%
- **Timeline**: 产品完善 → 自用1个月 → 评估效果
- **Qualitative**: 不再脑记任务、跟着排程执行、焦虑降低
