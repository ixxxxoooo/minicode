package projects

import (
	"encoding/json"
	"errors"
	"fmt"
)

// WorkspaceFileExt 是工作区文件(双击用 MiniCode 打开)的扩展名。
const WorkspaceFileExt = ".mcworkspace"

// MarshalWorkspaceFile 把工作区写成 .mcworkspace 的内容。ID 与权限确认记录不带出去:
// 对方机器上要重新分配 ID,附加目录的权限也要在那边重新确认。
func MarshalWorkspaceFile(w Workspace) ([]byte, error) {
	w.ID = ""
	w.PermSaveAck, w.PermSessionAck = false, false
	return json.MarshalIndent(w, "", "  ")
}

// ParseWorkspaceFile 读 .mcworkspace 的内容;至少要有一个带路径的文件夹。
func ParseWorkspaceFile(b []byte) (Workspace, error) {
	var w Workspace
	if err := json.Unmarshal(b, &w); err != nil {
		return w, errors.New("不是有效的工作区文件")
	}
	var fs []Folder
	for _, f := range w.Folders {
		if f.Path != "" {
			fs = append(fs, f)
		}
	}
	if len(fs) == 0 {
		return w, errors.New("工作区文件里没有文件夹")
	}
	if len(fs) > MaxFolders {
		return w, fmt.Errorf("工作区文件夹太多(最多 %d 个)", MaxFolders)
	}
	w.Folders = fs
	w.ID = ""
	w.PermSaveAck, w.PermSessionAck = false, false
	return w, nil
}
