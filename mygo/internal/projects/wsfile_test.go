package projects

import "testing"

func TestWorkspaceFileRoundTrip(t *testing.T) {
	w := Workspace{ID: "ws-1", Name: "全栈", Folders: []Folder{{Path: "/a", Name: "前端"}, {Path: "/b"}}, LoadMD: true, PermSaveAck: true}
	b, err := MarshalWorkspaceFile(w)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseWorkspaceFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "" || got.PermSaveAck || got.PermSessionAck {
		t.Fatalf("ID 与权限确认不应带出: %+v", got)
	}
	if got.Name != "全栈" || len(got.Folders) != 2 || got.Folders[0].Name != "前端" || !got.LoadMD {
		t.Fatalf("内容不一致: %+v", got)
	}
	for _, bad := range []string{"hello", "{}", `{"folders":[{"path":""}]}`} {
		if _, err := ParseWorkspaceFile([]byte(bad)); err == nil {
			t.Fatalf("应报错: %q", bad)
		}
	}
}
