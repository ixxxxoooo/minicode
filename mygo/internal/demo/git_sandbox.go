package demo

import "fmt"

// gitSandboxExtraFiles 追加到 router-demo 沙盒的文件(与 git 演示状态对应).
func gitSandboxExtraFiles() map[string]string {
	files := gitRouterWorkExtras()
	files["src/legacy.rs"] = "pub fn legacy() {}\n"
	files["src/cache.rs"] = gitMultiHunkWork()
	files["Cargo.lock"] = "# lock v2\n"
	files["demo/conflicts/two.rs"] = conflictTwo
	files["demo/conflicts/five.toml"] = conflictFive
	files["demo/conflicts/broken.md"] = conflictBroken
	files["demo/conflicts/binary.dat"] = "BINARY_V2\n"
	files["demo/conflicts/both_added.json"] = conflictBothAdded
	files["demo/conflicts/delete_modify.txt"] = conflictDeleteModify
	for i := 1; i <= 300; i++ {
		p := fmt.Sprintf("demo/bulk/file_%03d.txt", i)
		if _, ok := files[p]; !ok {
			if i%73 == 0 {
				continue
			}
			files[p] = fmt.Sprintf("work %d\n", i)
		}
	}
	return files
}
