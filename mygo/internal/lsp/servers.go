package lsp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Spec 描述一种语言服务器:服务哪些扩展名、可执行文件名(按优先级)、启动参数、没装时的安装办法。
type Spec struct {
	ID      string
	Name    string   // 显示名,如「rust-analyzer」
	Lang    string   // 语言名,如「Rust」
	Exts    []string // 含点,如 ".rs"
	LangIDs map[string]string
	Bins    []string
	Args    []string
	Install string   // 安装命令(在终端里运行)
	Hint    string   // 安装前提 / 备注
	Roots   []string // 项目根标记文件(向上找)
}

// Specs 是内置的服务器清单。
var Specs = []Spec{
	{ID: "rust", Name: "rust-analyzer", Lang: "Rust", Exts: []string{".rs"}, Bins: []string{"rust-analyzer"},
		Install: "rustup component add rust-analyzer", Hint: "需要 rustup", Roots: []string{"Cargo.toml"}},
	{ID: "go", Name: "gopls", Lang: "Go", Exts: []string{".go"}, Bins: []string{"gopls"},
		Install: "go install golang.org/x/tools/gopls@latest", Hint: "需要 Go 工具链", Roots: []string{"go.work", "go.mod"}},
	{ID: "python", Name: "Pyright", Lang: "Python", Exts: []string{".py", ".pyi"},
		Bins: []string{"pyright-langserver", "basedpyright-langserver"}, Args: []string{"--stdio"},
		Install: "npm install -g pyright", Hint: "需要 Node.js / npm", Roots: []string{"pyproject.toml", "setup.py", "requirements.txt", "pyrightconfig.json"}},
	{ID: "lua", Name: "lua-language-server", Lang: "Lua", Exts: []string{".lua"}, Bins: []string{"lua-language-server"},
		Install: "brew install lua-language-server", Hint: "需要 Homebrew", Roots: []string{".luarc.json", ".luarc.jsonc", ".stylua.toml"}},
	{ID: "ts", Name: "typescript-language-server", Lang: "TypeScript / JavaScript", Exts: []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"},
		LangIDs: map[string]string{".ts": "typescript", ".tsx": "typescriptreact", ".js": "javascript", ".jsx": "javascriptreact", ".mjs": "javascript", ".cjs": "javascript"},
		Bins:    []string{"typescript-language-server"}, Args: []string{"--stdio"},
		Install: "npm install -g typescript typescript-language-server", Hint: "需要 Node.js / npm", Roots: []string{"tsconfig.json", "package.json"}},
	{ID: "c", Name: "clangd", Lang: "C / C++", Exts: []string{".c", ".h", ".cc", ".cpp", ".hpp", ".cxx", ".m", ".mm"},
		LangIDs: map[string]string{".c": "c", ".h": "c", ".cc": "cpp", ".cpp": "cpp", ".hpp": "cpp", ".cxx": "cpp", ".m": "objective-c", ".mm": "objective-cpp"},
		Bins:    []string{"clangd"}, Install: "brew install llvm", Hint: "Xcode 命令行工具自带 clangd", Roots: []string{"compile_commands.json", "CMakeLists.txt", ".clangd"}},
	{ID: "swift", Name: "SourceKit-LSP", Lang: "Swift", Exts: []string{".swift"}, Bins: []string{"sourcekit-lsp"},
		Install: "xcode-select --install", Hint: "随 Xcode / Swift 工具链提供", Roots: []string{"Package.swift"}},
}

// ForFile 按扩展名选服务器。
func ForFile(path string) *Spec {
	ext := strings.ToLower(filepath.Ext(path))
	for i := range Specs {
		for _, e := range Specs[i].Exts {
			if e == ext {
				return &Specs[i]
			}
		}
	}
	return nil
}

// LangID 是 didOpen 的 languageId。
func (s *Spec) LangID(path string) string {
	if id, ok := s.LangIDs[strings.ToLower(filepath.Ext(path))]; ok {
		return id
	}
	return s.ID
}

// searchDirs 是 PATH 之外常见的安装目录。
func searchDirs() []string {
	home, _ := os.UserHomeDir()
	dirs := filepath.SplitList(os.Getenv("PATH"))
	dirs = append(dirs, "/opt/homebrew/bin", "/usr/local/bin", "/usr/bin")
	if home != "" {
		dirs = append(dirs, filepath.Join(home, "go", "bin"), filepath.Join(home, ".cargo", "bin"), filepath.Join(home, ".local", "bin"),
			filepath.Join(home, ".bun", "bin"))
		if m, _ := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin")); len(m) > 0 {
			dirs = append(dirs, m...)
		}
		// rustup 工具链里的真实 rust-analyzer(~/.cargo/bin 里的是转发到 rustup 的代理,没装组件时一启动就退出)
		if m, _ := filepath.Glob(filepath.Join(home, ".rustup", "toolchains", "*", "bin")); len(m) > 0 {
			dirs = append(m, dirs...)
		}
	}
	return dirs
}

// Find 找可执行文件的路径;找不到返回 "", false。
func (s *Spec) Find() (string, bool) {
	for _, bin := range s.Bins {
		for _, d := range searchDirs() {
			p := filepath.Join(d, bin)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
				if s.ID == "rust" && isRustupProxy(p) {
					continue
				}
				return p, true
			}
		}
		if p, err := exec.LookPath(bin); err == nil {
			return p, true
		}
	}
	return "", false
}

// isRustupProxy:~/.cargo/bin 里的 rust-analyzer 若是指向 rustup 的符号链接就是代理。
func isRustupProxy(p string) bool {
	dest, err := os.Readlink(p)
	return err == nil && filepath.Base(dest) == "rustup"
}

// Root 从文件所在目录向上找项目根标记,找不到用 fallback(项目目录)。
func (s *Spec) Root(file, fallback string) string {
	dir := filepath.Dir(file)
	for {
		for _, m := range s.Roots {
			if _, err := os.Stat(filepath.Join(dir, m)); err == nil {
				return dir
			}
		}
		if dir == fallback || dir == filepath.Dir(dir) || !HasPrefixPath(dir, fallback) {
			break
		}
		dir = filepath.Dir(dir)
	}
	return fallback
}

// Start 启动服务器进程并完成握手。stderr 的内容经 logf 送出(进「输出 → 语言服务」)。
func (s *Spec) Start(bin, root string, h Handlers, logf func(string)) (*Client, *exec.Cmd, error) {
	cmd := exec.Command(bin, s.Args...)
	cmd.Dir = root
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	go func() {
		buf := make([]byte, 4096)
		var carry string
		for {
			n, err := stderr.Read(buf)
			if n > 0 && logf != nil {
				carry += string(buf[:n])
				for {
					i := strings.IndexByte(carry, '\n')
					if i < 0 {
						break
					}
					if ln := strings.TrimSpace(carry[:i]); ln != "" {
						logf(ln)
					}
					carry = carry[i+1:]
				}
			}
			if err != nil {
				return
			}
		}
	}()
	c := NewClient(stdout, stdin, h)
	go func() { <-c.Done(); _ = cmd.Wait() }()
	if err := c.Initialize(PathToURI(root), filepath.Base(root), 60*time.Second); err != nil {
		_ = cmd.Process.Kill()
		return nil, nil, err
	}
	return c, cmd, nil
}
