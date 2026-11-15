package claude

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 假 claude:和真实行为一致——stream-json 没带 --verbose 时只在 stderr 报错、退出码为 0;
// 带了才输出 init + 助手文本。自动化必须带 --verbose,且「退出 0 但无输出」要当失败。
func fakeStreamClaude(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	fake := filepath.Join(dir, "claude")
	if err := os.WriteFile(fake, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MINI_CLAUDE_PATH", fake)
}

func TestAutomationStreamNeedsVerbose(t *testing.T) {
	fakeStreamClaude(t, `#!/bin/sh
case " $* " in
*" --verbose "*) ;;
*) echo "Error: When using --print, --output-format=stream-json requires --verbose" >&2; exit 0;;
esac
echo '{"type":"system","subtype":"init","session_id":"sess-1"}'
echo '{"type":"assistant","message":{"content":[{"type":"text","text":"没有未使用的函数"}]}}'
echo '{"type":"result","result":"没有未使用的函数"}'
`)
	res, err := RunAutomationPromptStreamCtx(context.Background(), t.TempDir(), "检查", "plan", 5*time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.SessionID != "sess-1" || !strings.Contains(res.Output, "没有未使用的函数") {
		t.Fatalf("结果不对:%+v", res)
	}
	if n := strings.Count(res.Output, "没有未使用的函数"); n != 1 {
		t.Fatalf("assistant 与 result 是同一段文本,不能重复:出现 %d 次:%q", n, res.Output)
	}
}

func TestAutomationStreamEmptyOutputIsFailure(t *testing.T) {
	fakeStreamClaude(t, "#!/bin/sh\necho 'something went wrong' >&2\nexit 0\n")
	_, err := RunAutomationPromptStreamCtx(context.Background(), t.TempDir(), "检查", "plan", 5*time.Second, nil)
	if err == nil || !strings.Contains(err.Error(), "something went wrong") {
		t.Fatalf("退出 0 但无输出应报错并带 stderr:%v", err)
	}
}
