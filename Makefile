# TickTask Makefile

.PHONY: all dev prod build build-backend build frontend clean test help

# 默认目标
all: build

# 开发模式启动
dev:
	@./scripts/start.sh dev

# 生产模式启动
prod:
	@./scripts/start.sh prod

# 构建 (后端 + 前端)
build: build-backend build-frontend

# 仅构建后端
build-backend:
	@./scripts/build.sh backend

# 仅构建前端
build-frontend:
	@./scripts/build.sh frontend

# 安装依赖
install:
	@echo "安装后端依赖..."
	cd backend && go mod download
	@echo "安装前端依赖..."
	cd frontend && npm install
	@echo "✅ 依赖安装完成"

# 运行测试
test:
	@echo "运行后端测试..."
	cd backend && go test ./internal/... -v
	@echo "运行前端构建检查..."
	cd frontend && npm run build

# 清理构建产物
clean:
	@echo "清理构建产物..."
	rm -rf backend/bin
	rm -rf frontend/dist
	@echo "✅ 清理完成"

# 显示帮助
help:
	@echo "TickTask 命令"
	@echo ""
	@echo "  make dev           开发模式启动"
	@echo "  make prod          生产模式启动"
	@echo "  make build         构建后端和前端"
	@echo "  make build-backend 仅构建后端"
	@echo "  make build-frontend 仅构建前端"
	@echo "  make install       安装依赖"
	@echo "  make test          运行测试"
	@echo "  make clean         清理构建产物"