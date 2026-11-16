package customize

import (
	"testing"
	"time"
)

// OpenDisk 不能再起 claude 子进程取市场列表(它会在界面线程里被调用):
// 即使 PATH 里有一个会睡很久的假 claude,打开也应当瞬间完成,且市场是兜底数据。
func TestOpenDiskDoesNotRunMarketplaceCLI(t *testing.T) {
	_, proj := setupDiskHome(t)
	t0 := time.Now()
	ds, err := OpenDisk(proj)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(t0); d > 500*time.Millisecond {
		t.Fatalf("OpenDisk 耗时 %v,疑似又在同步起子进程", d)
	}
	if len(ds.Market) == 0 || ds.Market[0].Name != "frontend-design" {
		t.Fatalf("应使用兜底市场数据:%+v", ds.Market)
	}
}

// SetMarket:替换为真实列表,已安装的插件重新标注;空列表保持原样。
func TestSetMarketKeepsInstalledFlag(t *testing.T) {
	_, proj := setupDiskHome(t)
	ds, err := OpenDisk(proj)
	if err != nil {
		t.Fatal(err)
	}
	ds.items[Plugins] = append(ds.items[Plugins], &Item{Name: "real-plug"})
	ds.SetMarket(nil)
	if ds.Market[0].Name != "frontend-design" {
		t.Fatal("空列表不应替换")
	}
	ds.SetMarket([]*Market{{Name: "real-plug"}, {Name: "other"}})
	if len(ds.Market) != 2 || !ds.Market[0].Installed || ds.Market[1].Installed {
		t.Fatalf("已安装标注不对:%+v %+v", ds.Market[0], ds.Market[1])
	}
}
