package web

import (
	"os"
	"testing"
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

func TestDistFSContainsIndexHTML(t *testing.T) {
	f, err := DistFS().Open("index.html")
	if err != nil {
		t.Fatalf("open index.html: %v", err)
	}
	f.Close()
}

func TestIsStubTrueInRepoState(t *testing.T) {
	// 仓库默认只提交占位页；构建脚本打完 exe 会恢复占位页，此断言保持确定。
	if !IsStub() {
		t.Fatal("IsStub() = false, want true (repo embeds the placeholder)")
	}
}

func TestFindDiskDistEmptyOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	if got := FindDiskDist(); got != "" {
		t.Fatalf("FindDiskDist() = %q, want empty", got)
	}
}
