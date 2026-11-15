package vcs

// 提交图谱的泳道布局(纯函数):给定按「新 → 旧」排列的提交,算出每一行怎么画。
// 做法是顺着时间往下走,维护「各泳道此刻在等哪个提交」:
//   - 提交落在等它的泳道上(没有就开新泳道);多条泳道都在等它时,其余泳道在这里汇入(Join);
//   - 第一个父提交沿用本泳道,其余父提交:已有泳道在等它就并过去(Merge),否则开新泳道(Fork)。

// GraphRow 是一行的泳道画法。泳道下标从 0 起,越小越靠左。
type GraphRow struct {
	Node   int   // 提交节点所在泳道
	Top    []int // 从上边缘进入本行的泳道(含节点泳道,如果它之前就存在)
	Bottom []int // 从下边缘离开本行的泳道
	Join   []int // 在本行汇入节点的其它泳道(曲线:泳道顶 → 节点)
	Merge  []int // 节点向已有泳道并过去(曲线:节点 → 泳道底)
	Fork   []int // 节点向新开的泳道分出去(曲线:节点 → 泳道底)
	Width  int   // 本行用到的最大泳道数
}

// Layout 计算全部行。
func Layout(commits []Commit) []GraphRow {
	var lanes []string // lanes[i] = 泳道 i 正在等的提交哈希,空串 = 空闲
	rows := make([]GraphRow, 0, len(commits))
	for _, c := range commits {
		var r GraphRow
		for i, h := range lanes {
			if h != "" {
				r.Top = append(r.Top, i)
			}
		}
		// 节点泳道:第一个在等它的;没有就开新的
		node := -1
		for i, h := range lanes {
			if h == c.Hash {
				if node < 0 {
					node = i
				} else {
					r.Join = append(r.Join, i)
					lanes[i] = ""
				}
			}
		}
		if node < 0 {
			node = freeLane(&lanes)
		}
		r.Node = node
		// 父提交
		lanes[node] = ""
		for k, p := range c.Parents {
			if k == 0 {
				lanes[node] = p
				continue
			}
			at := -1
			for i, h := range lanes {
				if h == p {
					at = i
					break
				}
			}
			if at >= 0 {
				r.Merge = append(r.Merge, at)
			} else {
				at = freeLane(&lanes)
				lanes[at] = p
				r.Fork = append(r.Fork, at)
			}
		}
		// 第一个父提交若已被别的泳道等着(两条线指向同一个祖先):本泳道并过去
		if len(c.Parents) > 0 {
			for i, h := range lanes {
				if i != node && h == c.Parents[0] {
					r.Merge = append(r.Merge, i)
					lanes[node] = ""
					break
				}
			}
		}
		for i, h := range lanes {
			if h != "" {
				r.Bottom = append(r.Bottom, i)
			}
		}
		r.Width = len(lanes)
		for len(lanes) > 0 && lanes[len(lanes)-1] == "" { // 收掉右侧空泳道
			lanes = lanes[:len(lanes)-1]
		}
		rows = append(rows, r)
	}
	return rows
}

func freeLane(lanes *[]string) int {
	for i, h := range *lanes {
		if h == "" {
			return i
		}
	}
	*lanes = append(*lanes, "")
	return len(*lanes) - 1
}
