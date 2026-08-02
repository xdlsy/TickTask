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
- `POST /api/tasks` - 创建任务
- `PUT /api/tasks/:id` - 更新任务
- `DELETE /api/tasks/:id` - 删除任务
- `PATCH /api/tasks/:id/move` - 移动任务到其他象限

### 计时器
- `GET /api/sessions/active` - 获取当前活跃会话
- `GET /api/sessions/recent` - 获取最近会话
- `POST /api/sessions` - 创建并启动会话
- `PATCH /api/sessions/:id/control` - 控制会话

### AI 智能功能
- `GET /api/ai/status` - 获取 AI 配置状态
- `POST /api/ai/classify` - 单任务智能分类
- `POST /api/ai/classify/batch` - 批量任务分类
- `POST /api/ai/schedule` - 生成每日日程
- `GET /api/ai/priority` - 获取优先级建议

### 设置
- `GET /api/settings` - 获取设置
- `PUT /api/settings/pomodoro` - 更新番茄时钟设置
- `PUT /api/settings/ai` - 更新 AI 设置

### WebSocket
- `WS /ws` - 实时连接（计时器状态推送）

## 开发计划

- [x] Go 后端项目初始化
- [x] Vue 3 前端项目初始化
- [x] 数据库模型与迁移
- [x] 基础 API 框架搭建
- [x] 任务 CRUD 功能实现
- [x] 番茄计时器功能实现
- [x] 四象限视图实现
- [x] AI 智能分类集成

## 许可证

MIT
