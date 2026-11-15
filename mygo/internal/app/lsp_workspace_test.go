package app

import "testing"

func TestLspWorkspaceScanBumpsGen(t *testing.T) {
	a := New(Options{Palette: "paper", Scheme: "a"})
	a.SetProject(t.TempDir())
	a.prefs.Set("lsp.workspaceDiag", true)
	gen := a.lspH.workspaceGen
	a.lspWorkspaceScan()
	if a.lspH.workspaceGen != gen+1 {
		t.Fatalf("启动扫描应递增 workspaceGen: %d -> %d", gen, a.lspH.workspaceGen)
	}
}