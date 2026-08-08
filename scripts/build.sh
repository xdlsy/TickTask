#!/bin/bash

# TickTask 构建脚本
# 用于编译后端和构建前端

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "🔨 TickTask 构建脚本"
echo "======================"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 构建后端
build_backend() {
    echo -e "${YELLOW}📦 构建后端...${NC}"
    cd "$PROJECT_DIR/backend"

    # 检查 Go 是否安装
    if ! command -v go &> /dev/null; then
        echo -e "${RED}错误: Go 未安装，请先安装 Go${NC}"
        exit 1
    fi

    # 设置 Go 代理（中国大陆加速）
    export GOPROXY=https://goproxy.cn,direct

    # 下载依赖
    echo "下载 Go 依赖..."
    go mod download

    # 编译
    echo "编译后端..."
    CGO_ENABLED=1 go build -o bin/ticktask-server cmd/server/main.go

    echo -e "${GREEN}✅ 后端构建完成: backend/bin/ticktask-server${NC}"
}

# 构建前端
build_frontend() {
    echo -e "${YELLOW}📦 构建前端...${NC}"
    cd "$PROJECT_DIR/frontend"

    # 检查 Node.js 是否安装
    if ! command -v node &> /dev/null; then
        echo -e "${RED}错误: Node.js 未安装，请先安装 Node.js${NC}"
        exit 1
    fi

    # 检查 npm 是否安装
    if ! command -v npm &> /dev/null; then
        echo -e "${RED}错误: npm 未安装，请先安装 npm${NC}"
        exit 1
    fi

    # 安装依赖
    if [ ! -d "node_modules" ]; then
        echo "安装 npm 依赖..."
        npm install
    fi

    # 构建
    echo "构建前端..."
    npm run build

    echo -e "${GREEN}✅ 前端构建完成: frontend/dist/${NC}"
}

# 主函数
main() {
    case "${1:-all}" in
        backend)
            build_backend
            ;;
        frontend)
            build_frontend
            ;;
        all)
            build_backend
            build_frontend
            ;;
        *)
            echo "用法: $0 [backend|frontend|all]"
            exit 1
            ;;
    esac

    echo ""
    echo -e "${GREEN}🎉 构建完成！${NC}"
    echo ""
    echo "运行以下命令启动应用:"
    echo "  ./scripts/start.sh"
}

main "$@"