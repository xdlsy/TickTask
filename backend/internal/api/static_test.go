package api

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
}

// 仓库默认嵌入占位页：无磁盘 dist 时兜底返回占位页，API 未匹配返回 404 JSON。
// 注意：依赖"仓库默认只含占位页"这一不变式（构建脚本打完 exe 会恢复占位页）。
func TestServeFrontendFallbackServesStub(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	chdir(t, dir)

	r := gin.New()
	serveFrontend(r)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Fatalf("GET / code = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ticktask-frontend-stub") {
		t.Fatalf("GET / body missing stub marker, got: %.200s", w.Body.String())
	}

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest("GET", "/api/nope", nil))
	if w2.Code != 404 {
		t.Fatalf("GET /api/nope code = %d, want 404", w2.Code)
	}
}
