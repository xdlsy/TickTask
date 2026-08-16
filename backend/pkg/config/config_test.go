package config

import (
	"os"
	"path/filepath"
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

// 同时设置 Windows(APPDATA) 与 Linux(XDG_CONFIG_HOME) 的用户配置目录
func setUserConfigDir(t *testing.T, dir string) {
	t.Helper()
	for _, k := range []string{"APPDATA", "XDG_CONFIG_HOME"} {
		old, had := os.LookupEnv(k)
		os.Setenv(k, dir)
		t.Cleanup(func() {
			if had {
				os.Setenv(k, old)
			} else {
				os.Unsetenv(k)
			}
		})
	}
}

func unsetUserConfigDir(t *testing.T) {
	t.Helper()
	for _, k := range []string{"APPDATA", "XDG_CONFIG_HOME"} {
		old, had := os.LookupEnv(k)
		os.Unsetenv(k)
		t.Cleanup(func() {
			if had {
				os.Setenv(k, old)
			} else {
				os.Unsetenv(k)
			}
		})
	}
}

func writeConfig(t *testing.T, path, portYAML string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "server:\n  port: " + portYAML + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolvePrefersCwdConfig(t *testing.T) {
	cwd := t.TempDir()
	appdata := t.TempDir()
	writeConfig(t, filepath.Join(cwd, "configs", "config.yaml"), "9999")
	writeConfig(t, filepath.Join(appdata, "TickTask", "config.yaml"), "7777")
	chdir(t, cwd)
	setUserConfigDir(t, appdata)

	cfg, path := Resolve()
	if cfg.Server.Port != 9999 {
		t.Fatalf("port = %d, want 9999 (CWD config wins)", cfg.Server.Port)
	}
	if path != filepath.Join("configs", "config.yaml") {
		t.Fatalf("path = %q, want configs/config.yaml", path)
	}
}

func TestResolveFallsBackToAppDirConfig(t *testing.T) {
	cwd := t.TempDir()
	appdata := t.TempDir()
	writeConfig(t, filepath.Join(appdata, "TickTask", "config.yaml"), "7777")
	chdir(t, cwd)
	setUserConfigDir(t, appdata)

	cfg, path := Resolve()
	if cfg.Server.Port != 7777 {
		t.Fatalf("port = %d, want 7777 (APPDATA config)", cfg.Server.Port)
	}
	want := filepath.Join(appdata, "TickTask", "config.yaml")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestResolveDefaultsPutDataUnderAppDir(t *testing.T) {
	cwd := t.TempDir()
	appdata := t.TempDir()
	chdir(t, cwd)
	setUserConfigDir(t, appdata)

	cfg, path := Resolve()
	if path != "" {
		t.Fatalf("path = %q, want empty (defaults)", path)
	}
	want := filepath.Join(appdata, "TickTask", "data", "ticktask.db")
	if cfg.Database.Path != want {
		t.Fatalf("db path = %q, want %q", cfg.Database.Path, want)
	}
}

func TestResolveDefaultsFallbackWhenNoUserDir(t *testing.T) {
	cwd := t.TempDir()
	chdir(t, cwd)
	unsetUserConfigDir(t)

	cfg, path := Resolve()
	if path != "" {
		t.Fatalf("path = %q, want empty", path)
	}
	if cfg.Database.Path != "./data/ticktask.db" {
		t.Fatalf("db path = %q, want ./data/ticktask.db", cfg.Database.Path)
	}
}

func TestAppDirJoinsTickTask(t *testing.T) {
	base := t.TempDir()
	setUserConfigDir(t, base)
	dir, ok := AppDir()
	if !ok {
		t.Fatal("ok = false, want true")
	}
	want := filepath.Join(base, "TickTask")
	if dir != want {
		t.Fatalf("dir = %q, want %q", dir, want)
	}
}
