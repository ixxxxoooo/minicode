package update

import (
	"sync"
	"testing"
)

func collect(r *Runner) (*[]Progress, *sync.Mutex) {
	var mu sync.Mutex
	var got []Progress
	r.OnProgress = func(p Progress) {
		mu.Lock()
		got = append(got, p)
		mu.Unlock()
	}
	return &got, &mu
}

// CheckOnly:发现新版本只上报 Found,不下载;之后 Download 才继续。
func TestRunnerCheckOnlyThenDownload(t *testing.T) {
	r := &Runner{Fetch: DemoFetcher{Version: "2.0.0"}, CurrentVer: "1.0.0", CheckOnly: true, workDir: t.TempDir()}
	got, _ := collect(r)
	r.Run(t.Context())
	last := (*got)[len(*got)-1]
	if !last.Found || last.Version != "2.0.0" {
		t.Fatalf("应以 Found 结束:%+v", last)
	}
	for _, p := range *got {
		if p.Phase == PhaseDownload {
			t.Fatalf("CheckOnly 不该下载:%+v", p)
		}
	}
	if r.FoundVersion() != "2.0.0" {
		t.Fatalf("应记住已发现版本,得到 %q", r.FoundVersion())
	}
	*got = nil
	r.Download(t.Context())
	if end := (*got)[len(*got)-1]; end.Phase != PhaseReady || end.Version != "2.0.0" {
		t.Fatalf("Download 应走到 Ready:%+v", end)
	}
	if r.FoundVersion() != "" {
		t.Fatal("Download 后不该再留着已发现版本")
	}
}

// 没有已发现版本就 Download:给出 check 阶段的可读错误,而不是静默或 panic。
func TestRunnerDownloadWithoutFound(t *testing.T) {
	r := &Runner{Fetch: DemoFetcher{}, CurrentVer: "1.0.0", workDir: t.TempDir()}
	got, _ := collect(r)
	r.Download(t.Context())
	if len(*got) != 1 || (*got)[0].Err == "" || (*got)[0].ErrStage != "check" {
		t.Fatalf("应报一条 check 阶段错误:%+v", *got)
	}
}

// 不是 CheckOnly:行为与之前一致,直接下载到 Ready。
func TestRunnerAutoDownloadUnchanged(t *testing.T) {
	r := &Runner{Fetch: DemoFetcher{Version: "2.0.0"}, CurrentVer: "1.0.0", workDir: t.TempDir()}
	got, _ := collect(r)
	r.Run(t.Context())
	if end := (*got)[len(*got)-1]; end.Phase != PhaseReady || end.Found {
		t.Fatalf("默认应直接下载到 Ready:%+v", end)
	}
}
