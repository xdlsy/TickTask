# TickTask

一款个人时间管理工具，集成番茄工作法、四象限法则和 AI 智能推荐。

## 功能特性

| 功能 | 描述 |
|-----|------|
| 番茄时钟 | 基于番茄工作法的专注计时器 |
| 四象限法排序 | 按重要性和紧急度对任务分类 |
| 事务整理 | 任务创建、编辑、删除、状态管理 |
| 智能时间安排 | 基于大模型 AI 的任务优先级和时间推荐 |

## 技术架构

### 后端 (Go)
- Gin Web 框架
- GORM ORM
- SQLite 数据库
- WebSocket 实时推送

### 前端 (Vue 3)
- Vue 3 + TypeScript
- Element Plus UI 组件
- Pinia 状态管理
- Vue Router 路由

## 快速开始

### 一键启动（推荐）

```bash
# 开发模式（前后端分离，支持热重载）
make dev
# 或
./scripts/start.sh dev

# 生产模式（单端口，后端服务前端静态文件）
make prod
# 或
./scripts/start.sh prod
```

开发模式访问 `http://localhost:5173`
生产模式访问 `http://localhost:8080`

### 手动启动

#### 后端启动

```bash
cd backend

# 安装依赖
go mod download

# 运行
go run cmd/server/main.go
```

#### 前端启动

```bash
cd frontend

# 安装依赖
npm install

# 运行开发服务器
npm run dev
```

访问 `http://localhost:5173` 即可使用。

### 构建命令

```bash
# 构建后端和前端
make build
# 或
./scripts/build.sh all

# 仅构建后端
make build-backend
# 或
./scripts/build.sh backend

# 仅构建前端
make build-frontend
# 或
./scripts/build.sh frontend
```

## 目录结构

```
TickTask/
├── backend/          # Go 后端
│   ├── cmd/server/   # 程序入口
│   ├── internal/     # 业务逻辑
│   │   ├── api/      # API 处理
│   │   ├── service/  # 服务层
│   │   ├── repository/ # 数据访问
│   │   ├── model/    # 数据模型
│   │   ├── ai/       # AI 模块
│   │   └── websocket/ # WebSocket
│   ├── pkg/          # 公共包
│   └── data/         # SQLite 数据库
├── frontend/         # Vue 前端
│   ├── src/api/      # API 调用
│   ├── src/components/ # 组件
│   ├── src/views/    # 页面
│   ├── src/stores/   # Pinia 状态
│   └── src/utils/    # 工具函数
└── scripts/          # 启动和构建脚本
    ├── build.sh      # 构建脚本
    └── start.sh      # 启动脚本
```

## 配置

后端配置文件：`backend/configs/config.yaml`

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"

ai:
  provider: "openai"
  api_key: ""  # 设置你的 API Key
  model: "gpt-4o-mini"
```

## API 端点

### 任务
- `GET /api/tasks` - 获取任务列表
- `GET /api/tasks/quadrant` - 获取按象限分组的任务
- `GET /api/tasks/:id` - 获取单个任务
- `POST /api/tasks` - 创建任务
- `PUT /api/tasks/:id` - 更新任务
- `DELETE /api/tasks/:id` - 删除任务
- `PATCH /api/tasks/:id/move` - 移动任务到其他象限

### 计时器
- `GET /api/sessions/active` - 获取当前活跃会话
- `GET /api/sessions/recent` - 获取最近会话
- `GET /api/sessions/today-stats` - 获取今日番茄-任务统计
- `POST /api/sessions` - 创建并启动会话
- `PATCH /api/sessions/:id/control` - 控制会话（暂停/恢复/停止）

### AI 智能功能
- `GET /api/ai/status` - 获取 AI 配置状态
- `POST /api/ai/classify` - 单任务智能分类
- `POST /api/ai/classify/batch` - 批量任务分类
- `POST /api/ai/classify-task-text` - 按文本分类（未建任务前）
- `POST /api/ai/schedule` - 生成每日日程
- `POST /api/ai/reschedule-after-interrupt` - 打断后重新排程
- `GET /api/ai/priority` - 获取优先级建议
- `GET /api/ai/daily-insights` - 获取每日 AI 洞察

### 设置
- `GET /api/settings` - 获取设置
- `PUT /api/settings/pomodoro` - 更新番茄时钟设置
- `PUT /api/settings/ai` - 更新 AI 设置

### 日程
- `GET /api/schedules` - 获取日程列表
- `GET /api/schedules/:id` - 获取单个日程
- `POST /api/schedules` - 创建日程
- `PUT /api/schedules/:id` - 更新日程
- `DELETE /api/schedules/:id` - 删除日程
- `PUT /api/schedules/:id/move` - 移动日程
- `POST /api/schedules/generate` - AI 生成日程
- `POST /api/schedules/revise` - AI 修订日程（预览变更）
- `POST /api/schedules/revise/apply` - 应用修订
- `DELETE /api/schedules` - 清空所有日程

### 数据分析
- `GET /api/analytics/summary` - 每日汇总
- `GET /api/analytics/trend` - 趋势数据
- `GET /api/analytics/distribution` - 象限/任务分布
- `GET /api/analytics/pomodoro-by-task` - 按任务的番茄统计
- `GET /api/analytics/pomodoro-trends` - 番茄趋势

### 工作日志
- `GET /api/work-logs/today/context` - 今日上下文（用于 AI 拆条）
- `POST /api/work-logs/structure` - AI 将脑暴拆成结构化 items
- `GET /api/work-logs` - 日报列表
- `POST /api/work-logs` - 创建日报
- `GET /api/work-logs/:date` - 获取某日日报
- `PUT /api/work-logs/:date` - 更新日报
- `PATCH /api/work-logs/:date/summary` - 更新日报摘要
- `POST /api/work-logs/:date/items` - 新增快捷录入 item
- `PATCH /api/work-logs/:date/items/:itemId` - 更新快捷录入 item
- `DELETE /api/work-logs/:date/items/:itemId` - 删除快捷录入 item

### 工作日志周期报告
- `POST /api/work-reports/generate` - 生成周期报告（周/月/半年/年）
- `GET /api/work-reports` - 报告列表
- `GET /api/work-reports/:type/:periodKey` - 获取指定周期报告

### 数据管理
- `GET /api/data/export` - 导出全部数据（JSON 备份）
- `POST /api/data/import/preview` - 导入预览（冲突检测）
- `POST /api/data/import/apply` - 应用导入（按策略 + 逐条覆盖）
- `DELETE /api/data/all` - 清空全部数据（保留配置）

### WebSocket
- `WS /ws` - 实时连接（计时器状态推送）

## 开发计划

- [x] Go 后端项目初始化
- [x] Vue 3 前端项目初始化
- [x] 数据库模型与迁移
- [x] 基础 API 框架搭建
- [x] 任务 CRUD 功能实现
- [x] 番茄计时器功能实现（WebSocket 实时推送）
- [x] 四象限视图实现
- [x] AI 智能分类 / 优先级 / 日程生成
- [x] 日程管理（日/周/月视图 + AI 修订工作流）
- [x] 循环任务与偏好时段
- [x] 使用分析（专注时长 / 完成率 / 象限分布 / 番茄趋势）
- [x] 工作日志（脑暴 → AI 拆条 → 日报 → 周期报告）
- [x] 数据导入导出与清空（按模块冲突解决 + 原子清空）

## 许可证

MIT
