// Package web 嵌入前端构建产物。仓库默认只提交占位页；打包脚本在编译前
// 把 frontend/dist 拷贝到本目录，编译后再恢复占位页。
package web

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
)

//go:embed all:dist
var distFS embed.FS

const stubMarker = "ticktask-frontend-stub"

// DistFS 返回嵌入的前端文件系统（内容即 dist/ 目录）。
func DistFS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err) // dist 始终随二进制嵌入，不会失败
	}
	return sub
}

// IsStub 报告嵌入的前端是否为占位页（即构建前未拷入真实产物）。
func IsStub() bool {
	b, err := fs.ReadFile(distFS, "dist/index.html")
	if err != nil {
		return true
	}
	return bytes.Contains(b, []byte(stubMarker))
}

// FindDiskDist 返回工作目录下首个存在的前端磁盘 dist（仓库开发布局），
// 都不存在时返回 ""（打包 exe 场景）。
func FindDiskDist() string {
	for _, p := range []string{"dist", "../frontend/dist", "../../frontend/dist"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
