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
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/ticktask-server ./cmd/server

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

# 构建单文件 exe：前端产物嵌入二进制，目标机器无需 Go/Node/gcc
build_exe() {
    echo -e "${YELLOW}📦 构建单文件 exe...${NC}"

    # 1) 构建前端
    "$SCRIPT_DIR/build.sh" frontend

    # 2) 拷贝前端产物到嵌入目录（覆盖占位页）
    echo "拷贝前端产物到 backend/web/dist ..."
    rm -rf "$PROJECT_DIR/backend/web/dist"
    cp -r "$PROJECT_DIR/frontend/dist" "$PROJECT_DIR/backend/web/dist"

    # 3) 纯 Go 构建（无 CGO）
    cd "$PROJECT_DIR/backend"
    export GOPROXY=https://goproxy.cn,direct
    go mod download
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/ticktask.exe ./cmd/server
    cd "$PROJECT_DIR"

    # 4) 恢复占位页：保证 go test 的 stub 语义与 git 状态干净
    rm -rf "$PROJECT_DIR/backend/web/dist"
    git -C "$PROJECT_DIR" checkout -- backend/web/dist

    echo -e "${GREEN}✅ 单文件 exe 构建完成: backend/bin/ticktask.exe${NC}"
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
        exe)
            build_exe
            ;;
        *)
            echo "用法: $0 [backend|frontend|all|exe]"
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