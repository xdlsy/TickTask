# TickTask Backend

Go 后端服务，提供时间管理功能的 API。

## 开发

```bash
# 安装依赖
go mod download

# 运行
go run cmd/server/main.go
```

## 构建

```bash
go build -o bin/ticktask cmd/server/main.go
```

## API 端点

- `GET /api/tasks` - 获取任务列表
- `GET /api/tasks/quadrant` - 获取按象限分组的任务
- `POST /api/tasks` - 创建任务
- `PUT /api/tasks/:id` - 更新任务
- `DELETE /api/tasks/:id` - 删除任务
- `PATCH /api/tasks/:id/move` - 移动任务到其他象限

- `GET /api/sessions/active` - 获取当前活跃会话
- `GET /api/sessions/recent` - 获取最近会话
- `POST /api/sessions` - 创建并启动会话
- `PATCH /api/sessions/:id/control` - 控制会话

- `GET /ws` - WebSocket 连接
