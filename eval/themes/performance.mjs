import {
  called, pending, succeeded, failed, anyWrite, anyDangerous, noTool,
  tools, toolNames, argsOf, argOf, txt, has,
  declined, askedClarify, mentionedEmpty, notFabricated, askedConfirm,
  today, shiftDays, weekRange, daysBetween,
} from '../lib/helpers.mjs';

// Performance/SLO test cases - all require timing infrastructure
export const CASES = [
  {
    cat: 'performance',
    prompt: '我今天的安排',
    check: () => [true, 'needs timing: <10s end-to-end'],
    note: 'needs-timing'
  },
  {
    cat: 'performance',
    prompt: `我需要创建一个非常重要的项目任务，这个任务涉及到整个团队的核心业务流程优化和系统重构工作。具体来说，我们需要重新设计现有的客户关系管理系统，包括前端用户界面的现代化改造，后端API的性能优化，以及数据库架构的深度调整。这个项目的目标是提升用户体验，提高系统响应速度，并增强数据安全性和可扩展性。

项目的主要工作内容包括：首先，我们需要对现有系统进行全面的代码审查和性能分析，识别出所有可能的性能瓶颈和安全隐患。然后，设计新的系统架构，采用微服务模式来解耦各个功能模块，使用分布式缓存来提升读取性能，引入消息队列来处理异步任务。前端方面，我们将采用最新的React框架和TypeScript来实现类型安全，使用现代化的UI组件库来提升用户界面的一致性和美观度。

在数据库层面，我们需要从单一的关系型数据库迁移到混合架构，保留关系型数据库来处理事务性操作，同时引入文档型数据库来处理非结构化数据存储需求。我们还计划实现数据仓库和分析平台，支持实时的业务数据分析和报表生成。

项目的时间规划非常紧张，我们需要在接下来的三个月内完成所有核心功能的开发和测试工作。团队规模约为15人，包括前端开发工程师、后端开发工程师、数据库管理员、DevOps工程师以及产品经理和测试工程师。我们需要采用敏捷开发方法，每两周一个迭代，确保持续交付和用户反馈的快速整合。

此外，这个项目还需要考虑多个合规性要求，包括GDPR数据保护法规、PCI-DSS支付卡行业安全标准，以及公司内部的数据治理政策。我们需要在整个开发过程中嵌入安全最佳实践，实施全面的代码审查和安全测试。

项目的成功关键指标包括：系统响应时间提升至少50%，并发用户数支持提升3倍，系统可用性达到99.9%，用户满意度评分提升20%以上。我们需要建立完善的监控和告警体系，确保能够及时发现和解决生产环境中的问题。

这个项目对公司的发展至关重要，它不仅能够显著提升我们的技术实力和市场竞争力，还能为未来的产品创新和技术演进奠定坚实的基础。因此，我们需要全力以赴，确保项目按时按质完成。建个任务`,
    check: () => [true, 'needs timing: <30s long-input'],
    note: 'needs-timing'
  },
  {
    cat: 'performance',
    prompt: '列出今天所有安排并按时间排序',
    check: () => [true, 'needs timing: <20s multi-tool'],
    note: 'needs-timing'
  },
  {
    cat: 'performance',
    prompt: '我有哪些任务',
    check: () => [true, 'needs timing: <10s simple'],
    note: 'needs-timing'
  }
];
