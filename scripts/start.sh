#!/bin/bash

# TickTask 启动脚本
# 支持开发模式和生产模式

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认端口
BACKEND_PORT=${BACKEND_PORT:-8080}
FRONTEND_PORT=${FRONTEND_PORT:-5173}

# 清理函数
cleanup() {
    echo ""
    echo -e "${YELLOW}正在停止服务...${NC}"
    if [ ! -z "$BACKEND_PID" ]; then
        kill $BACKEND_PID 2>/dev/null || true
    fi
    if [ ! -z "$FRONTEND_PID" ]; then
        kill $FRONTEND_PID 2>/dev/null || true
    fi
    echo -e "${GREEN}服务已停止${NC}"
    exit 0
}

trap cleanup SIGINT SIGTERM

# 开发模式
start_dev() {
    echo -e "${BLUE}🚀 启动开发模式...${NC}"
    echo ""

    # 设置 Go 代理（中国大陆加速）
    export GOPROXY=https://goproxy.cn,direct

    # 启动后端
    echo -e "${YELLOW}启动后端服务 (端口: $BACKEND_PORT)...${NC}"
    cd "$PROJECT_DIR/backend"
    if [ ! -f "go.mod" ]; then
        echo -e "${RED}错误: 后端目录缺少 go.mod${NC}"
        exit 1
    fi
    go run cmd/server/main.go &
    BACKEND_PID=$!
    cd "$PROJECT_DIR"

    # 等待后端启动
    sleep 2

    # 启动前端
    echo -e "${YELLOW}启动前端开发服务器 (端口: $FRONTEND_PORT)...${NC}"
    cd "$PROJECT_DIR/frontend"
    if [ ! -f "package.json" ]; then
        echo -e "${RED}错误: 前端目录缺少 package.json${NC}"
        exit 1
    fi

    # 检查是否需要安装依赖
    if [ ! -d "node_modules" ]; then
        echo "安装前端依赖..."
        npm install
    fi

    npm run dev &
    FRONTEND_PID=$!
    cd "$PROJECT_DIR"

    echo ""
    echo -e "${GREEN}✅ 服务启动成功！${NC}"
    echo ""
    echo -e "${BLUE}访问地址:${NC}"
    echo "  前端: http://localhost:$FRONTEND_PORT"
    echo "  后端: http://localhost:$BACKEND_PORT"
    echo ""
    echo -e "${YELLOW}按 Ctrl+C 停止服务${NC}"
    echo ""

    # 等待子进程
    wait
}

# 生产模式
start_prod() {
    echo -e "${BLUE}🚀 启动生产模式...${NC}"
    echo ""

    # 检查后端二进制文件
    BACKEND_BIN="$PROJECT_DIR/backend/bin/ticktask-server"
    if [ ! -f "$BACKEND_BIN" ]; then
        echo -e "${YELLOW}后端二进制文件不存在，正在构建...${NC}"
        "$SCRIPT_DIR/build.sh" backend
    fi

    # 检查前端构建产物
    FRONTEND_DIST="$PROJECT_DIR/frontend/dist"
    if [ ! -d "$FRONTEND_DIST" ]; then
        echo -e "${YELLOW}前端构建产物不存在，正在构建...${NC}"
        "$SCRIPT_DIR/build.sh" frontend
    fi

    # 启动后端（后端会自动服务前端静态文件）
    echo -e "${YELLOW}启动后端服务 (端口: $BACKEND_PORT)...${NC}"
    cd "$PROJECT_DIR/backend"
    ./bin/ticktask-server &
    BACKEND_PID=$!
    cd "$PROJECT_DIR"

    echo ""
    echo -e "${GREEN}✅ 服务启动成功！${NC}"
    echo ""
    echo -e "${BLUE}访问地址:${NC}"
    echo "  应用: http://localhost:$BACKEND_PORT"
    echo ""
    echo -e "${YELLOW}按 Ctrl+C 停止服务${NC}"
    echo ""

    # 等待子进程
    wait
}

# 帮助信息
show_help() {
    echo "TickTask 启动脚本"
    echo ""
    echo "用法: $0 [命令] [选项]"
    echo ""
    echo "命令:"
    echo "  dev       开发模式启动 (默认)"
    echo "  prod      生产模式启动"
    echo "  build     构建应用"
    echo "  help      显示帮助信息"
    echo ""
    echo "环境变量:"
    echo "  BACKEND_PORT   后端端口 (默认: 8080)"
    echo "  FRONTEND_PORT  前端端口 (默认: 5173, 仅开发模式)"
    echo ""
    echo "示例:"
    echo "  $0              # 开发模式启动"
    echo "  $0 dev          # 开发模式启动"
    echo "  $0 prod         # 生产模式启动"
    echo "  BACKEND_PORT=3000 $0 prod  # 指定端口启动"
}

# 主函数
main() {
    case "${1:-dev}" in
        dev)
            start_dev
            ;;
        prod)
            start_prod
            ;;
        build)
            "$SCRIPT_DIR/build.sh" all
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo -e "${RED}未知命令: $1${NC}"
            show_help
            exit 1
            ;;
    esac
}

main "$@"