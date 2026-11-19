package lsp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Spec 描述一种语言服务器:服务哪些扩展名、可执行文件名(按优先级)、启动参数、没装时的安装办法。
type Spec struct {
	ID      string
	Name    string   // 显示名,如「rust-analyzer」
	Lang    string   // 语言名,如「Rust」
	Exts    []string // 含点,如 ".rs"
	Names   []string // 没有扩展名的文件名,如 "Dockerfile"
	LangIDs map[string]string
	Bins    []string
	Args    []string
	Install string   // 安装命令(在终端里运行)
	Hint    string   // 安装前提 / 备注
	Roots   []string // 项目根标记文件(向上找)

	// 托管安装(npm 系的服务器):NPM 是固定版本的包(如 "pyright@1.1.414"),空 = 不支持下载安装;
	// NPMGroup 让同一个包提供的多个服务器共用一个目录;MinNode 是它要求的最低 Node 版本(本机 Node 不够就用托管的)。
	NPM      []string
	NPMGroup string
	MinNode  string
}

// Specs 是内置的服务器清单。
var Specs = []Spec{
	{ID: "rust", Name: "rust-analyzer", Lang: "Rust", Exts: []string{".rs"}, Bins: []string{"rust-analyzer"},
		Install: "rustup component add rust-analyzer", Hint: "需要 rustup", Roots: []string{"Cargo.toml"}},
	{ID: "go", Name: "gopls", Lang: "Go", Exts: []string{".go"}, Bins: []string{"gopls"},
		Install: "go install golang.org/x/tools/gopls@latest", Hint: "需要 Go 工具链", Roots: []string{"go.work", "go.mod"}},
	{ID: "python", Name: "Pyright", Lang: "Python", Exts: []string{".py", ".pyi"},
		Bins: []string{"pyright-langserver", "basedpyright-langserver"}, Args: []string{"--stdio"},
		NPM: []string{"pyright@1.1.414"}, MinNode: "18.0.0", Install: "npm install -g pyright", Hint: "需要 Node.js / npm", Roots: []string{"pyproject.toml", "setup.py", "requirements.txt", "pyrightconfig.json"}},
	{ID: "lua", Name: "lua-language-server", Lang: "Lua", Exts: []string{".lua"}, Bins: []string{"lua-language-server"},
		Install: "brew install lua-language-server", Hint: "需要 Homebrew", Roots: []string{".luarc.json", ".luarc.jsonc", ".stylua.toml"}},
	{ID: "ts", Name: "typescript-language-server", Lang: "TypeScript / JavaScript", Exts: []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"},
		LangIDs: map[string]string{".ts": "typescript", ".tsx": "typescriptreact", ".js": "javascript", ".jsx": "javascriptreact", ".mjs": "javascript", ".cjs": "javascript"},
		Bins:    []string{"typescript-language-server"}, Args: []string{"--stdio"},
		NPM: []string{"typescript@7.0.2", "typescript-language-server@6.1.0"}, MinNode: "22.22.2", Install: "npm install -g typescript typescript-language-server", Hint: "需要 Node.js / npm", Roots: []string{"tsconfig.json", "package.json"}},
	{ID: "c", Name: "clangd", Lang: "C / C++", Exts: []string{".c", ".h", ".cc", ".cpp", ".hpp", ".cxx", ".m", ".mm"},
		LangIDs: map[string]string{".c": "c", ".h": "c", ".cc": "cpp", ".cpp": "cpp", ".hpp": "cpp", ".cxx": "cpp", ".m": "objective-c", ".mm": "objective-cpp"},
		Bins:    []string{"clangd"}, Install: "brew install llvm", Hint: "Xcode 命令行工具自带 clangd", Roots: []string{"compile_commands.json", "CMakeLists.txt", ".clangd"}},
	{ID: "swift", Name: "SourceKit-LSP", Lang: "Swift", Exts: []string{".swift"}, Bins: []string{"sourcekit-lsp"},
		Install: "xcode-select --install", Hint: "随 Xcode / Swift 工具链提供", Roots: []string{"Package.swift"}},
	{ID: "java", Name: "jdtls", Lang: "Java", Exts: []string{".java"}, Bins: []string{"jdtls"},
		Install: "brew install jdtls", Hint: "需要 JDK 17+(brew install openjdk)", Roots: []string{"pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle", ".project"}},
	{ID: "kotlin", Name: "kotlin-language-server", Lang: "Kotlin", Exts: []string{".kt", ".kts"}, Bins: []string{"kotlin-language-server"},
		Install: "brew install kotlin-language-server", Hint: "需要 JDK 17+", Roots: []string{"settings.gradle.kts", "settings.gradle", "build.gradle.kts", "build.gradle", "pom.xml"}},
	{ID: "csharp", Name: "csharp-ls", Lang: "C#", Exts: []string{".cs", ".csx"}, Bins: []string{"csharp-ls"},
		Install: "dotnet tool install --global csharp-ls", Hint: "需要 .NET SDK;装完把 ~/.dotnet/tools 加进 PATH", Roots: []string{"*.sln", "*.csproj", "global.json"}},
	{ID: "php", Name: "Intelephense", Lang: "PHP", Exts: []string{".php", ".phtml"}, Bins: []string{"intelephense"}, Args: []string{"--stdio"},
		NPM: []string{"intelephense@1.18.5"}, MinNode: "18.0.0", Install: "npm install -g intelephense", Hint: "需要 Node.js / npm", Roots: []string{"composer.json", ".git"}},
	{ID: "ruby", Name: "ruby-lsp", Lang: "Ruby", Exts: []string{".rb", ".rake", ".gemspec", ".erb"}, Bins: []string{"ruby-lsp"},
		LangIDs: map[string]string{".erb": "erb"},
		Install: "gem install ruby-lsp", Hint: "需要 Ruby / RubyGems", Roots: []string{"Gemfile", ".ruby-version"}},
	{ID: "html", Name: "vscode-html-language-server", Lang: "HTML", Exts: []string{".html", ".htm"}, Bins: []string{"vscode-html-language-server"}, Args: []string{"--stdio"},
		NPM: []string{"vscode-langservers-extracted@4.10.0"}, NPMGroup: "vscode-langservers", MinNode: "18.0.0", Install: "npm install -g vscode-langservers-extracted", Hint: "需要 Node.js / npm;同一个包也提供 CSS / JSON", Roots: []string{"package.json", "index.html"}},
	{ID: "css", Name: "vscode-css-language-server", Lang: "CSS / SCSS / Less", Exts: []string{".css", ".scss", ".less"}, Bins: []string{"vscode-css-language-server"}, Args: []string{"--stdio"},
		LangIDs: map[string]string{".css": "css", ".scss": "scss", ".less": "less"},
		NPM:     []string{"vscode-langservers-extracted@4.10.0"}, NPMGroup: "vscode-langservers", MinNode: "18.0.0", Install: "npm install -g vscode-langservers-extracted", Hint: "需要 Node.js / npm;同一个包也提供 HTML / JSON", Roots: []string{"package.json"}},
	{ID: "json", Name: "vscode-json-language-server", Lang: "JSON", Exts: []string{".json", ".jsonc"}, Bins: []string{"vscode-json-language-server"}, Args: []string{"--stdio"},
		LangIDs: map[string]string{".json": "json", ".jsonc": "jsonc"},
		NPM:     []string{"vscode-langservers-extracted@4.10.0"}, NPMGroup: "vscode-langservers", MinNode: "18.0.0", Install: "npm install -g vscode-langservers-extracted", Hint: "需要 Node.js / npm;同一个包也提供 HTML / CSS", Roots: []string{"package.json"}},
	{ID: "yaml", Name: "yaml-language-server", Lang: "YAML", Exts: []string{".yaml", ".yml"}, Bins: []string{"yaml-language-server"}, Args: []string{"--stdio"},
		NPM: []string{"yaml-language-server@2.0.0"}, MinNode: "22.13.0", Install: "npm install -g yaml-language-server", Hint: "需要 Node.js / npm"},
	{ID: "toml", Name: "taplo", Lang: "TOML", Exts: []string{".toml"}, Bins: []string{"taplo"}, Args: []string{"lsp", "stdio"},
		Install: "brew install taplo", Hint: "需要 Homebrew(或 cargo install taplo-cli --features lsp)"},
	{ID: "sql", Name: "sqls", Lang: "SQL", Exts: []string{".sql"}, Bins: []string{"sqls"},
		Install: "go install github.com/sqls-org/sqls@latest", Hint: "需要 Go 工具链;补全 / 诊断要在项目里放 .sqls/config.yml 配数据库连接,没配时只有语法层面的功能"},
	{ID: "bash", Name: "bash-language-server", Lang: "Shell", Exts: []string{".sh", ".bash", ".zsh"}, Bins: []string{"bash-language-server"}, Args: []string{"start"},
		LangIDs: map[string]string{".sh": "shellscript", ".bash": "shellscript", ".zsh": "shellscript"},
		NPM:     []string{"bash-language-server@5.8.1"}, MinNode: "20.0.0", Install: "npm install -g bash-language-server", Hint: "需要 Node.js / npm;装 shellcheck(brew install shellcheck)后诊断更全"},
	{ID: "docker", Name: "docker-langserver", Lang: "Dockerfile", Exts: []string{".dockerfile"}, Names: []string{"Dockerfile", "Containerfile"},
		Bins: []string{"docker-langserver"}, Args: []string{"--stdio"}, LangIDs: map[string]string{"": "dockerfile", ".dockerfile": "dockerfile"},
		NPM: []string{"dockerfile-language-server-nodejs@0.15.0"}, MinNode: "18.0.0", Install: "npm install -g dockerfile-language-server-nodejs", Hint: "需要 Node.js / npm"},
	{ID: "terraform", Name: "terraform-ls", Lang: "Terraform", Exts: []string{".tf", ".tfvars"}, Bins: []string{"terraform-ls"}, Args: []string{"serve"},
		LangIDs: map[string]string{".tf": "terraform", ".tfvars": "terraform-vars"},
		Install: "brew install hashicorp/tap/terraform-ls", Hint: "需要 Homebrew", Roots: []string{".terraform", "main.tf"}},
	{ID: "svelte", Name: "svelte-language-server", Lang: "Svelte", Exts: []string{".svelte"}, Bins: []string{"svelteserver"}, Args: []string{"--stdio"},
		NPM: []string{"svelte-language-server@0.18.6"}, MinNode: "18.0.0", Install: "npm install -g svelte-language-server", Hint: "需要 Node.js / npm", Roots: []string{"svelte.config.js", "svelte.config.ts", "package.json"}},
	{ID: "dart", Name: "Dart 分析服务", Lang: "Dart / Flutter", Exts: []string{".dart"}, Bins: []string{"dart"}, Args: []string{"language-server", "--protocol=lsp"},
		Install: "brew install dart-sdk", Hint: "Flutter 自带 dart;需要 Dart SDK", Roots: []string{"pubspec.yaml"}},
	{ID: "zig", Name: "zls", Lang: "Zig", Exts: []string{".zig", ".zon"}, Bins: []string{"zls"},
		Install: "brew install zls", Hint: "需要 Homebrew", Roots: []string{"build.zig"}},
	{ID: "elixir", Name: "elixir-ls", Lang: "Elixir", Exts: []string{".ex", ".exs", ".heex"}, Bins: []string{"elixir-ls"},
		LangIDs: map[string]string{".ex": "elixir", ".exs": "elixir", ".heex": "phoenix-heex"},
		Install: "brew install elixir-ls", Hint: "需要 Elixir / Erlang", Roots: []string{"mix.exs"}},
	{ID: "haskell", Name: "haskell-language-server", Lang: "Haskell", Exts: []string{".hs", ".lhs"}, Bins: []string{"haskell-language-server-wrapper"}, Args: []string{"--lsp"},
		Install: "ghcup install hls", Hint: "需要 ghcup(https://www.haskell.org/ghcup)", Roots: []string{"stack.yaml", "cabal.project", "hie.yaml", "package.yaml"}},
	{ID: "latex", Name: "texlab", Lang: "LaTeX", Exts: []string{".tex", ".bib", ".sty", ".cls"}, Bins: []string{"texlab"},
		LangIDs: map[string]string{".tex": "latex", ".sty": "latex", ".cls": "latex", ".bib": "bibtex"},
		Install: "brew install texlab", Hint: "需要 Homebrew;编译另需 TeX 发行版", Roots: []string{".latexmkrc", "latexmkrc"}},
}

// ForFile 按扩展名选服务器。
func ForFile(path string) *Spec {
	ext := strings.ToLower(filepath.Ext(path))
	base := filepath.Base(path)
	for i := range Specs {
		for _, e := range Specs[i].Exts {
			if e == ext {
				return &Specs[i]
			}
		}
		for _, n := range Specs[i].Names {
			if n == base || strings.HasPrefix(base, n+".") { // Dockerfile / Dockerfile.dev
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
	if len(s.Names) > 0 && s.LangIDs[""] != "" {
		return s.LangIDs[""]
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
		dirs = append(dirs, nvmBins(home)...) // 新版本在前(Glob 是字典序,v9 会排到 v24 后面)
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
	return s.ManagedBin() // 用户自己装的都没有,再用应用下载安装的
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
			if strings.ContainsAny(m, "*?[") { // 通配,如 *.sln / *.csproj
				if g, _ := filepath.Glob(filepath.Join(dir, m)); len(g) > 0 {
					return dir
				}
				continue
			}
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

// toolEnv 给语言服务器进程的环境:PATH 里先放可执行文件自己所在的目录,再放 searchDirs(本进程 PATH + 常见安装目录)。
//
// 为什么:从访达 / Dock 启动的应用只继承 launchd 的 PATH(/usr/bin:/bin:…),看不到 nvm / homebrew 的目录。
// 用 npm 装的服务器(intelephense、pyright、typescript-language-server…)开头是 `#!/usr/bin/env node`,
// 找不到 node 就以 127 立刻退出,界面只看到「语言服务器已退出」。npm 全局安装时 node 与服务器在同一个 bin 目录,
// 所以把它所在目录放最前面,既能找到 node,也保证是装它时用的那一版。
func (s *Spec) toolEnv(bin string) []string {
	var dirs []string
	seen := map[string]bool{}
	add := func(d string) {
		if d != "" && !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}
	add(filepath.Dir(bin))
	if IsManagedPath(bin) { // 托管目录的 .bin 里没有 node:加上选中的那个(本机优先,其次托管下载的)
		if n, ok := ResolveNode(s.MinNode, false); ok {
			add(n.Dir)
		}
	}
	for _, d := range searchDirs() {
		add(d)
	}
	path := "PATH=" + strings.Join(dirs, string(os.PathListSeparator))
	env := os.Environ()
	out := make([]string, 0, len(env)+1)
	for _, kv := range env {
		if !strings.HasPrefix(kv, "PATH=") {
			out = append(out, kv)
		}
	}
	return append(out, path)
}

// Start 启动服务器进程并完成握手。stderr 的内容经 logf 送出(进「输出 → 语言服务」);
// 握手失败时把 stderr 最后几行拼进错误(否则只剩一句「语言服务器已退出」,看不出为什么)。
func (s *Spec) Start(bin, root string, h Handlers, logf func(string)) (*Client, *exec.Cmd, error) {
	cmd := exec.Command(bin, s.Args...)
	cmd.Dir = root
	cmd.Env = s.toolEnv(bin)
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
	var (
		tailMu     sync.Mutex
		tail       []string // stderr 最后几行,握手失败时拼进错误
		stderrDone = make(chan struct{})
	)
	go func() {
		defer close(stderrDone)
		buf := make([]byte, 4096)
		var carry string
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				carry += string(buf[:n])
				for {
					i := strings.IndexByte(carry, '\n')
					if i < 0 {
						break
					}
					if ln := strings.TrimSpace(carry[:i]); ln != "" {
						if logf != nil {
							logf(ln)
						}
						tailMu.Lock()
						if tail = append(tail, ln); len(tail) > 4 {
							tail = tail[1:]
						}
						tailMu.Unlock()
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
		select { // 给 stderr 一点时间读完,错误里带上服务器自己说的话
		case <-stderrDone:
		case <-time.After(500 * time.Millisecond):
		}
		tailMu.Lock()
		why := strings.Join(tail, " | ")
		tailMu.Unlock()
		if r := []rune(why); len(r) > 300 {
			why = string(r[:300]) + "…"
		}
		if why != "" {
			err = fmt.Errorf("%w(服务器输出:%s)", err, why)
		}
		return nil, nil, err
	}
	return c, cmd, nil
}
