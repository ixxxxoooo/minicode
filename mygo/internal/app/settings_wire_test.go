package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ixxxxoooo/minicode/internal/claude"
	"github.com/ixxxxoooo/minicode/internal/core"
	"github.com/ixxxxoooo/minicode/internal/diag"
	"github.com/ixxxxoooo/minicode/internal/prefs"
)

func TestMigrateModelPrefs(t *testing.T) {
	st := prefs.Memory()
	st.Set("models.model", 1)
	a := &App{prefs: st}
	a.MigrateSettingsPrefs()
	if a.prefs.Str(prefModel, "") != "opus" {
		t.Fatalf("应迁移到 claude.model=opus,得 %q", a.prefs.Str(prefModel, ""))
	}
}

func TestFillClaudeConfigArgv(t *testing.T) {
	st := prefs.Memory()
	st.Set(pChatSys, "每次回复都以【测试】开头")
	st.Set(pRuntimeBin, "")
	st.Set(pPermMode, 2)
	st.Set(pPermDirs, "/tmp/extra-dir")
	st.Set(pRuntimeTO, "90")
	st.Set(pProxyMode, 2)
	st.Set(pProxyURL, "http://127.0.0.1:7890")
	st.Set("models.compact", true)
	a := &App{prefs: st}
	cfg := &claude.Config{Cwd: t.TempDir()}
	a.FillClaudeConfig(cfg)
	if cfg.AppendSystemPrompt != "每次回复都以【测试】开头" {
		t.Fatalf("追加系统提示不对:%q", cfg.AppendSystemPrompt)
	}
	if len(cfg.AddDirs) == 0 || cfg.AddDirs[0] != "/tmp/extra-dir" {
		t.Fatalf("--add-dir 应含额外目录:%v", cfg.AddDirs)
	}
	o := claude.Options{
		Cwd:                cfg.Cwd,
		AppendSystemPrompt: cfg.AppendSystemPrompt,
		AddDirs:            cfg.AddDirs,
		Mode:               cfg.Mode,
		ExtraEnv:           cfg.Env,
	}
	args := strings.Join(o.Args(), " ")
	for _, want := range []string{
		"--append-system-prompt",
		"【测试】",
		"--permission-mode plan",
		"--add-dir /tmp/extra-dir",
		"--autocompact auto",
	} {
		if !strings.Contains(args, want) {
			t.Errorf("参数缺少 %q:%s", want, args)
		}
	}
	var gotTimeout, gotProxy bool
	for _, e := range o.Env() {
		if strings.HasPrefix(e, "API_TIMEOUT_MS=") && strings.Contains(e, "90000") {
			gotTimeout = true
		}
		if strings.HasPrefix(e, "HTTP_PROXY=http") {
			gotProxy = true
		}
	}
	// ExtraEnv 在 Args 用 MINICODE_ 前缀,不会进入 o.Env() 合并前;RuntimeExtraEnv 在 FillClaudeConfig 已写入 cfg.Env
	for _, e := range cfg.Env {
		if e.Key == "API_TIMEOUT_MS" && e.Value == "90000" {
			gotTimeout = true
		}
		if e.Key == "HTTP_PROXY" && strings.Contains(e.Value, "7890") {
			gotProxy = true
		}
	}
	if !gotTimeout || !gotProxy {
		t.Fatalf("环境变量未接线:timeout=%v proxy=%v env=%v", gotTimeout, gotProxy, cfg.Env)
	}
	if cfg.Mode == nil || *cfg.Mode != core.ModePlan {
		t.Fatalf("默认权限模式应为计划:%v", cfg.Mode)
	}
}

func TestContextCompactThreshold(t *testing.T) {
	a := &App{prefs: prefs.Memory()}
	if a.ContextCompactThreshold() != 0.85 {
		t.Fatalf("默认阈值应为 0.85,得 %v", a.ContextCompactThreshold())
	}
	a.prefs.Set(pModelCmpAt, 70)
	if a.ContextCompactThreshold() != 0.7 {
		t.Fatalf("应为 0.7,得 %v", a.ContextCompactThreshold())
	}
}

func TestHandleComposerPaste(t *testing.T) {
	a := &App{prefs: prefs.Memory()}
	a.prefs.Set(pPasteFold, true)
	long := strings.Repeat("字", 2100)
	if !a.HandleComposerPaste(long) {
		t.Fatal("应折叠长粘贴")
	}
	if !strings.HasPrefix(a.ux.pendingInsert, "@") || !strings.HasSuffix(a.ux.pendingInsert, ".txt ") {
		t.Fatalf("折叠后应排队插入一条 @ 引用到正文:%q", a.ux.pendingInsert)
	}
}

func TestApplyDiagPrefsLogLevel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.json")
	p := prefs.Open(path)
	p.Set(pPrivacyLvl, 0)
	a := New(Options{Prefs: p})
	a.ApplyDiagPrefs()
	if diag.CurrentLevel() != diag.LevelError {
		t.Fatalf("want error, got %v", diag.CurrentLevel())
	}
}
