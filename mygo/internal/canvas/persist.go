package canvas

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// fileBlob 是磁盘上的单张画布文件。
type fileBlob struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Status   string    `json:"status"`
	Updated  time.Time `json:"updated"`
	Doc      Doc       `json:"doc"`
	Versions []Version `json:"versions"`
}

// Library 管理多张落盘画布(~/.minicode/canvases/)。
type Library struct {
	dir     string
	ids     []string // 按 Updated 降序
	active  string
	stores  map[string]*Store
	damaged map[string]string // id → 备份路径
}

// OpenLibrary 打开或创建画布目录。
func OpenLibrary(dir string) (*Library, error) {
	_ = os.MkdirAll(dir, 0o755)
	l := &Library{dir: dir, stores: map[string]*Store{}, damaged: map[string]string{}}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return l, err
	}
	type meta struct {
		id, title string
		at        time.Time
	}
	var list []meta
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var blob fileBlob
		if err := json.Unmarshal(b, &blob); err != nil {
			bad := filepath.Join(dir, e.Name()+".corrupt-"+fmt.Sprint(time.Now().Unix()))
			_ = os.Rename(filepath.Join(dir, e.Name()), bad)
			l.damaged[id] = bad
			continue
		}
		if blob.ID == "" {
			blob.ID = id
		}
		list = append(list, meta{id: blob.ID, title: blob.Title, at: blob.Updated})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].at.After(list[j].at) })
	for _, m := range list {
		l.ids = append(l.ids, m.id)
	}
	if len(l.ids) == 0 {
		return l, nil
	}
	l.active = l.ids[0]
	return l, nil
}

// IDs 返回画布 id 列表(最近修改在前)。
func (l *Library) IDs() []string { return append([]string(nil), l.ids...) }

// SeedDemo 把一张画布挂进库(不写盘,演示模式用)。
func (l *Library) SeedDemo(id string, s *Store) {
	if l.stores == nil {
		l.stores = map[string]*Store{}
	}
	l.stores[id] = s
	for _, x := range l.ids {
		if x == id {
			return
		}
	}
	l.ids = append(l.ids, id)
}

// Active 当前选中的画布 id。
func (l *Library) Active() string { return l.active }

// SetActive 切换当前画布。
func (l *Library) SetActive(id string) { l.active = id }

// Store 加载指定画布(带内存缓存)。
func (l *Library) Store(id string) (*Store, error) {
	if s, ok := l.stores[id]; ok {
		return s, nil
	}
	p := filepath.Join(l.dir, id+".json")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var blob fileBlob
	if err := json.Unmarshal(b, &blob); err != nil {
		bad := p + ".corrupt-" + fmt.Sprint(time.Now().Unix())
		_ = os.Rename(p, bad)
		return nil, err
	}
	s := &Store{Doc: blob.Doc, Versions: blob.Versions}
	if len(s.Versions) == 0 {
		s.Versions = []Version{{N: 1, By: "你", Note: "新建", At: blob.Updated, Doc: s.Doc.Clone()}}
	}
	l.stores[id] = s
	return s, nil
}

// ActiveStore 当前画布存储。
func (l *Library) ActiveStore() (*Store, error) {
	if l.active == "" {
		return nil, fmt.Errorf("没有画布")
	}
	return l.Store(l.active)
}

func (l *Library) save(id string, s *Store) error {
	blob := fileBlob{ID: id, Title: s.Doc.Title, Status: s.Doc.Status, Updated: time.Now(), Doc: s.Doc, Versions: s.Versions}
	b, err := json.MarshalIndent(blob, "", "  ")
	if err != nil {
		return err
	}
	p := filepath.Join(l.dir, id+".json")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Save 持久化指定画布。
func (l *Library) Save(id string) error {
	s, err := l.Store(id)
	if err != nil {
		return err
	}
	return l.save(id, s)
}

// SaveActive 保存当前画布。
func (l *Library) SaveActive() error {
	if l.active == "" {
		return nil
	}
	return l.Save(l.active)
}

// New 新建一张画布并落盘。
func (l *Library) New(title string, doc Doc, by, note string, at time.Time) (*Store, string, error) {
	id := slugID(title, at)
	if id == "" {
		id = fmt.Sprintf("canvas-%d", at.Unix())
	}
	s := NewStore(doc, by, note, at)
	l.stores[id] = s
	l.ids = append([]string{id}, l.ids...)
	l.active = id
	if err := l.save(id, s); err != nil {
		return s, id, err
	}
	return s, id, nil
}

// Rename 改标题(文件 id 不变)。
func (l *Library) Rename(id, title string) error {
	s, err := l.Store(id)
	if err != nil {
		return err
	}
	s.Doc.Title = title
	return l.save(id, s)
}

// Delete 删除一张画布。
func (l *Library) Delete(id string) error {
	delete(l.stores, id)
	for i, x := range l.ids {
		if x == id {
			l.ids = append(l.ids[:i], l.ids[i+1:]...)
			break
		}
	}
	if l.active == id {
		if len(l.ids) > 0 {
			l.active = l.ids[0]
		} else {
			l.active = ""
		}
	}
	return os.Remove(filepath.Join(l.dir, id+".json"))
}

func slugID(title string, at time.Time) string {
	var b strings.Builder
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-':
			b.WriteRune('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return ""
	}
	return strings.ToLower(s)
}

// maxImportBytes 导入的画布文件大小上限(防止误选巨大文件)。
const maxImportBytes = 8 << 20

// ErrNotCanvasFile 文件不是画布文件(不是合法 JSON,或没有画布内容)。
var ErrNotCanvasFile = fmt.Errorf("不是有效的画布文件")

// uniqueID 在库里找一个没被占用的 id(磁盘文件或内存里的都算),避免导入时覆盖已有画布。
func (l *Library) uniqueID(base string) string {
	taken := func(id string) bool {
		if _, ok := l.stores[id]; ok {
			return true
		}
		for _, x := range l.ids {
			if x == id {
				return true
			}
		}
		_, err := os.Stat(filepath.Join(l.dir, id+".json"))
		return err == nil
	}
	if !taken(base) {
		return base
	}
	for i := 2; ; i++ {
		id := fmt.Sprintf("%s-%d", base, i)
		if !taken(id) {
			return id
		}
	}
}

// Import 读取 .mccanvas 文件(与库里单张画布同一格式),作为新画布落盘并设为当前,返回新 id。
// 已有同名画布不会被覆盖,新画布换一个 id。
func (l *Library) Import(src string, at time.Time) (string, error) {
	st, err := os.Stat(src)
	if err != nil {
		return "", err
	}
	if st.Size() > maxImportBytes {
		return "", fmt.Errorf("文件太大(超过 %d MB)", maxImportBytes>>20)
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	var blob fileBlob
	if err := json.Unmarshal(b, &blob); err != nil {
		return "", ErrNotCanvasFile
	}
	if blob.Doc.Title == "" {
		blob.Doc.Title = blob.Title
	}
	if blob.Doc.Title == "" && len(blob.Doc.Blocks) == 0 && len(blob.Versions) == 0 {
		return "", ErrNotCanvasFile
	}
	if blob.Doc.Title == "" {
		blob.Doc.Title = strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	}
	base := slugID(blob.Doc.Title, at)
	if base == "" {
		base = fmt.Sprintf("canvas-%d", at.Unix())
	}
	id := l.uniqueID(base)
	s := &Store{Doc: blob.Doc, Versions: blob.Versions}
	if len(s.Versions) == 0 {
		s.Versions = []Version{{N: 1, By: "你", Note: "导入", At: at, Doc: s.Doc.Clone()}}
	}
	l.stores[id] = s
	l.ids = append([]string{id}, l.ids...)
	l.active = id
	if err := l.save(id, s); err != nil {
		return id, err
	}
	return id, nil
}

// Export 把一张画布写成 .mccanvas 文件(格式同库里单张画布),dest 的目录必须存在。
func (l *Library) Export(id, dest string) error {
	s, err := l.Store(id)
	if err != nil {
		return err
	}
	blob := fileBlob{ID: id, Title: s.Doc.Title, Status: s.Doc.Status, Updated: time.Now(), Doc: s.Doc, Versions: s.Versions}
	b, err := json.MarshalIndent(blob, "", "  ")
	if err != nil {
		return err
	}
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}
