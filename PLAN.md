# fuckHttp → Go + MyGo 重写计划

> 目标：把现有的 Rust/egui Windows 桌面程序 `fuckHttp` 用 Go + [mygo](https://github.com/egoist/mygo)（Native UI 模式）重写，功能对齐、行为可替换。
> 本文只做调研与方案，不含实现代码。

---

## 1. 现状分析

### 1.1 代码规模

| 文件 | 说明 |
|---|---|
| `src/main.rs` | 809 行，全部逻辑都在这一个文件里 |
| `Cargo.toml` | eframe/egui 0.27、reqwest(blocking)、regex、winreg、runas、is_elevated、dark-light、image |
| `build.rs` | winres 嵌入 icon.ico 与版本信息 |
| `icon.ico` | 图标资源 |

### 1.2 功能拆解（重写时必须逐项对齐）

| # | 功能 | Rust 位置 | 平台依赖 |
|---|---|---|---|
| 1 | 从拦截 URL 提取真实链接（QQ / 企业微信 / 通用参数 / fragment） | `extract_real_url_sync` | 无 |
| 2 | 微信安全页异步抓取 + 3 种正则 + base64 + redirecthelpcgi 兜底 | `extract_from_wechat_page` | 网络 |
| 3 | HTML 实体解码 | `decode_html_entities` | 无 |
| 4 | 枚举已安装浏览器 | `get_installed_browsers` | Windows 注册表 |
| 5 | 隐藏浏览器配置（%TEMP%\browser_selector_config.json） | `load_config`/`save_config` | 无 |
| 6 | 启动浏览器进程（解析命令行） | view 内联 | Windows 进程 |
| 7 | 无边框/置顶/居中/禁止缩放窗口，按浏览器数量自适应高度 | `main` + `update` | 窗口系统 |
| 8 | 标题栏：复制链接 / 设置 / 关闭 | `update` | 无 |
| 9 | URL 展示区（固定 60px，可滚动）+ 提取状态提示 | `update` | 无 |
| 10 | 设置页：浏览器显隐、注册/卸载、结果提示 | `update` | 注册表 |
| 11 | 注册为默认浏览器（Capabilities + RegisteredApplications + URL Protocol） | `register_app`/`unregister_app` | Windows 注册表 |
| 12 | 非管理员时提权执行 `--register`/`--unregister` | `runas` | Windows UAC |
| 13 | 点击窗口外部关闭（egui 版是"点击空白区域"） | `update` 末尾 | 窗口系统 |
| 14 | 复制成功 Toast | `toast_message` | 无 |
| 15 | 深/浅色跟随系统 | `dark_light::detect` | 系统主题 |
| 16 | 中文字体（读 C:\Windows\Fonts\msyh.ttc） | `BrowserSelectorApp::new` | Windows 字体 |
| 17 | 命令行：`fuckHttp.exe "<url>"`，无参默认 google.com | `main` | 无 |

### 1.3 必须保留的"外部契约"

这些是用户已经装在系统里的东西，改动会破坏升级路径：

- 注册表键名：`HKCU\Software\Classes\fuckHttp`、`HKLM\Software\fuckHttp\Capabilities`、`HKLM\Software\RegisteredApplications\fuckHttp`
- 配置文件名与位置：`%TEMP%\browser_selector_config.json`，字段 `hidden_browsers`
- 命令行参数：`--register` / `--unregister` / 首个非 `--` 参数为 URL
- 浏览器枚举来源：`SOFTWARE\Clients\StartMenuInternet`（HKLM + HKCU，按显示名去重）

---

## 2. 技术选型结论

### 2.1 MyGo 是什么（已核实）

`github.com/egoist/mygo`，MIT，Go 桌面应用框架：

- 两种窗口：**webview 页面** 或 **Native UI**（Go 直接用 `ui` 包画，GPU 渲染，无 webview）。
- **纯 Go，无 cgo**，任意平台交叉编译；单个二进制几 MB。
- 自带窗口/菜单/托盘/剪贴板/深链/文件关联/打包（Windows NSIS 安装包、macOS dmg、deb）。
- 版本：v0.7.4（proxy 上最新），**要求 Go 1.27.3+**。

### 2.2 环境核实结果

| 项 | 结果 | 结论 |
|---|---|---|
| 本机 Go | go1.26.5 linux/amd64 | 低于要求 |
| `GOTOOLCHAIN` | `auto` | 会自动下载 go1.27.3 |
| goproxy.cn 是否有 go1.27.3 工具链 | 有（`v0.0.1-go1.27.3.linux-amd64` 等） | 可自动升级，M0 需实测 |
| 本机 GTK3 | 有（`pkg-config gtk+-3.0` 通过） | Native UI 可在 Linux 本地跑，便于开发调试 |
| WebKitGTK | 有 | 本项目用不到（Native UI 无需 webview） |
| Bun / Node | 未安装 | Native UI 模板不需要，符合要求 |
| ImageMagick | 有（`magick`/`convert`） | 用于 icon.ico → icon.png |

### 2.3 选型结论

1. **Native UI 模式**（用户已确认），不引入前端构建链。
2. **Windows 优先**：注册表/提权部分用 build tag 隔离，非 Windows 提供桩实现，保证 Linux 上能 `go test` 和 `go tool mygo dev` 调试界面。
3. **不使用 mygo 的 `RequestSingleInstanceLock`**。原程序是"一次点击一个进程一个窗口"，加单实例会把第二个链接吞进第一个窗口，属于功能倒退。mygo 示例（launcher）用单实例是它的场景，这里不适用。
4. **不使用 mygo 的 `urlSchemes` / `fileAssociations`** 来做默认浏览器注册。它们只能注册自定义 scheme（`fuckhttp://`），无法注册 `http/https` 为默认浏览器；仍需手写注册表，与原实现一致。

---

## 3. 目标架构

### 3.1 目录（根目录即 Go 项目，Rust 版归档到 legacy/）

```
fuckHttp/
├── go.mod                              # module github.com/link-fgfgui/fuckHttp
├── mygo.json                           # 应用名/版本/图标
├── main.go                             # 入口：参数、注册模式、窗口、装配
├── resources/icon.png                  # 1024×1024，由 legacy/icon.ico 转换
├── internal/
│   ├── extract/                        # URL 提取（纯逻辑，可单测）
│   │   ├── extract.go
│   │   ├── wechat.go
│   │   ├── html.go
│   │   ├── extract_test.go
│   │   └── wechat_test.go
│   ├── browser/                        # 浏览器发现与启动
│   │   ├── browser.go
│   │   ├── launch_windows.go
│   │   ├── launch_linux.go
│   │   ├── launch_other.go
│   │   ├── registry_windows.go
│   │   ├── registry_linux.go
│   │   ├── registry_other.go
│   │   ├── browser_test.go
│   │   └── launch_linux_test.go
│   ├── config/config.go
│   ├── sysint/                         # 系统集成（注册/卸载/提权）
│   │   ├── register_windows.go
│   │   ├── register_linux.go
│   │   ├── register_linux_test.go
│   │   └── register_other.go
│   └── view/                           # 界面状态与视图
│       ├── app.go
│       ├── view.go
│       ├── view_test.go
│       └── render_test.go
├── legacy/                             # Rust 版源码（Cargo.toml、build.rs、src/、icon.ico）
├── PLAN.md                             # 本文件
└── README.md
```

依赖（全部纯 Go）：
```
require github.com/egoist/mygo v0.7.4
require golang.org/x/sys v0.x        // registry + windows API
tool    github.com/egoist/mygo/cmd/mygo
```

### 3.2 分层职责

- `extract`：无副作用，输入字符串 → 输出 `{URL, NeedAsync, Source}`；微信抓取单独函数，注入 `*http.Client` 便于测试。
- `browser`：`Browser{Name, Command, Hidden}`、`Installed()`、`Executable()`、`Launch(url)`。
- `config`：读写 `%TEMP%\browser_selector_config.json`，保持原字段名。
- `sysint`：注册表写入、提权。
- `view`：持有全部 UI 状态，`view(c *ui.Context)` 是纯渲染函数；副作用（网络、进程、注册表）全部走回调或 goroutine + `win.Update`。
- `main`：解析参数、处理 `--register`/`--unregister`、建窗口、接事件。

---

## 4. 模块设计要点

### 4.1 URL 提取（`internal/extract`）

```go
type Result struct {
    URL      string
    NeedAsync bool   // 微信页面需要异步抓取
    Source   string  // "qq" | "wecom" | "wechat" | "generic" | "none"
}

func Resolve(raw string) Result
func ResolveWeChat(ctx context.Context, c *http.Client, raw string) (string, error)
```

移植顺序与坑：

1. `url.Parse` 后取 `u.Hostname()`；用 `strings.Contains` 判断三个域名（与原 `host_str().contains` 等价）。
2. **双重解码**：Rust 的 `query_pairs()` 已经解过一次百分号，代码里又 `urlencoding::decode` 了一次。Go 的 `u.Query().Get()` 只解一次，所以必须再 `url.QueryUnescape` 一次，否则带 `%253A` 的链接会不一致。
3. 企业微信：`uri` 参数，无 scheme 时补 `https://`。
4. QQ：`url` 参数，必须已带 `http(s)://`。
5. 通用参数顺序：`url, link, target, redirect, goto, u, q`（保持原顺序，第一个命中即返回）。
6. fragment：直接判断 `http(s)://` 前缀。
7. 微信抓取：
   - 正则 1：`"name"\s*:\s*"继续访问",\s*"url"\s*:\s*"([^"]+)"`
   - 正则 2：`"desc"\s*:\s*"([^"]+)"`
   - 正则 3：`"name"\s*:\s*"申请恢复访问",\s*"url"\s*:\s*"([^"]+)"`
   - 正则 3 命中后：实体解码 → 解析 `url` 参数 → 标准 base64 解码 → 校验 http(s)；失败则把 `newreadtemplate` 换成 `redirecthelpcgi` 再请求 JSON，遍历 `btns[]` 找 `name=="申请恢复"`，取 `url` 参数 base64 解码。
   - `http.Client{Timeout: 10 * time.Second}`（原版无超时，会挂住协程）。
   - 建议加桌面 Chrome 的 `User-Agent`（原版是 reqwest 默认 UA，Go 默认 `Go-http-client/1.1` 更容易被拦）。**这是一处有意的行为改进，需在 M3 用真实链接验证。**
8. 实体解码：保持原 9 条替换，顺序不变。

### 4.2 浏览器发现（`internal/browser`）

```go
type Browser struct {
    Name    string `json:"name"`
    Command string `json:"command"`
    Hidden  bool   `json:"hidden"`
}

func Installed() ([]Browser, error)   // Windows 走注册表，其它平台返回桩
func (b Browser) Executable() string  // 命令行解析
func (b Browser) Launch(target string) error
```

- 枚举：`SOFTWARE\Clients\StartMenuInternet` → 子键 → 默认值为显示名 → `shell\open\command` 默认值为命令；HKLM 后 HKCU，按 Name 去重（与原逻辑一致）。
- 命令行解析（原样移植）：以 `"` 开头则取到下一个 `"`；否则取第一个空白前的 token。
- 启动：`exec.Command(exe, target)` + `SysProcAttr{HideWindow: true}`，`Start()` 后 `Process.Release()`，不等待。
- 已知缺陷（原版就有，**先保留**）：命令里的额外参数（如 `--single-argument %1`）被丢弃。作为可选项列在 §7。

### 4.3 配置（`internal/config`）

```go
type Config struct {
    HiddenBrowsers []string `json:"hidden_browsers"`
}
func Path() string  // filepath.Join(os.TempDir(), "browser_selector_config.json")
func Load() Config
func Save(Config) error
```

路径与 JSON 字段名保持原样，老用户配置可直接沿用。

### 4.4 系统集成（`internal/sysint`）

与 Rust 版逐键对齐：

```
HKCU\Software\Classes\fuckHttp
    (Default)                = "URL:fuckHttp Protocol"
    "URL Protocol"           = ""
    DefaultIcon\(Default)    = ""<exe>",0"
    shell\open\command\(Default) = ""<exe>" "%1""

HKLM\Software\fuckHttp\Capabilities
    ApplicationName        = "fuckHttp"
    ApplicationIcon        = ""<exe>",0"
    ApplicationDescription = "A custom browser selector."
    URLAssociations\http   = "fuckHttp"
    URLAssociations\https  = "fuckHttp"

HKLM\Software\RegisteredApplications
    fuckHttp = "Software\fuckHttp\Capabilities"
```

- 管理员判断：`windows.GetCurrentProcessToken().IsElevated()`。
- 提权：用 `github.com/arduino/go-windows-runas` v1.0.2（MIT）的 `RunElevated(exe, "", []string{arg}, true)`，它内部就是 `ShellExecuteExW` + `runas` + 等退出码，与原来手写的 `SHELLEXECUTEINFO` 版本等价，但不用自己维护 unsafe 结构体布局。管理员判断用它的 `IsAdminProcess()`。
- `--register`/`--unregister` 在 `mygo.App.Run()` **之前**处理，成功 `os.Exit(0)`，失败 `os.Exit(1)`。

### 4.5 窗口（`main.go`）

```go
win := mygo.NewWindow(mygo.WindowOptions{
    Title:         "fuckHttp",
    Width:         400,
    Height:        initialHeight,
    Frameless:     true,
    Transparent:   true,
    CornerRadius:  8,
    AlwaysOnTop:   true,
    DisableResize: true,
    Content:       ui.View(app.View),
})
win.OnBlur(win.Close)        // 点击窗口外部即关闭
```

高度公式原样移植（标题栏由视图绘制，计入窗口内高）：

```
h = 20 + (状态行 ? 20 : 0) + 60 + 20 + (可见浏览器数>0 ? 数量*50 + 20 : 0) + 25
设置页 h = 500
```

可见数量或状态变化时 `win.SetSize(400, h)`。

与原版差异（有意）：
- 原版"点击窗口内空白处关闭"→ 改为 `OnBlur`（真正的"点外部关闭"），语义更准确，也去掉了 100ms 防误触逻辑。
- 原版手工加载 msyh.ttc；mygo 用系统文本引擎（DirectWrite + 字体回退），中文与 emoji 自动处理，**删掉这段代码**。
- 主题默认跟随系统，无需 dark-light 探测。

### 4.6 视图（`internal/view`）

```go
type App struct {
    win       *mygo.Window
    url       string
    original  string
    browsers  []browser.Browser
    settings  bool
    message   string
    state     extractState   // idle / loading / ok / failed
    err       string
    copied    bool
}
```

结构：

1. 标题栏 `ui.Row(...).DragWindow()`：标题 + 复制 + 设置 + 关闭。
   - 图标按钮用 `ui.Icon(c, svg)`（`ui.MustParseSVG`），并加 `.Label("复制链接")` 保证可访问性；**不用 emoji**，避免字体回退差异。
2. 状态行：`ui.Spinner` + 文案（获取中 / 已提取 / 失败原因）。
3. URL 区：`ui.Scroll(c).Height(60)` 包 `ui.Text(c, url).Selectable()`。
4. `ui.Divider(c)`。
5. 浏览器按钮：`ui.Column` 默认横向拉伸，`ui.Button(c, b.Name).Height(40)`，点击 → `b.Launch(url)` → `win.Close()`。
6. 设置页：`ui.Switch(c, &b.Hidden).Label(b.Name)` + 名称；改动即 `config.Save`。注册/卸载按钮 + 结果行 + 当前注册状态。
7. 复制：`mygo.Clipboard.WriteText(url)` + `c.Toast("链接已复制到剪贴板")`。
8. 微信异步：`go extract.ResolveWeChat(...)` → `win.Update(func(){ ... })`。
9. Esc 关闭：`c.Shortcut(0, ui.KeyEscape)`（原版无此键，属低成本增强，可保留）。

### 4.7 入口（`main.go`）

```
1. 解析 os.Args
2. --register / --unregister → sysint → os.Exit
3. url := 第一个非 -- 参数，缺省 "https://www.google.com"
4. res := extract.Resolve(url)
5. browsers := browser.Installed() 应用 config 隐藏项
6. mygo.App.WhenReady(func(){ 建窗口; 需要异步则启动 goroutine })
7. mygo.App.Run()
```

**不加单实例锁**（见 §2.3）。

### 4.8 资源与打包

- `magick icon.ico -background none -resize 1024x1024 app/resources/icon.png`
- `mygo.json`：
  ```json
  {
    "name": "fuckHttp",
    "identifier": "com.iustech.fuckhttp",
    "version": "0.1.0",
    "copyright": "© 2025 iusTech.",
    "publisher": "iusTech."
  }
  ```
- 构建：`cd app && go tool mygo build -platform windows/amd64` → `build/windows-amd64/fuckHttp.exe` + NSIS 安装包（Linux 上加 `-skip-installer`）。

---

## 5. 与 Rust 版的行为差异（需明确认可）

| 项 | Rust 版 | Go 版 | 理由 |
|---|---|---|---|
| 关闭时机 | 点击窗口内空白区 | 窗口失焦（点外部） | 更符合直觉 |
| 中文/emoji 字体 | 手工加载 msyh.ttc | 系统文本引擎 + 回退 | 框架能力 |
| 微信请求超时 | 无（可能永久挂起） | 10s | 修复卡死 |
| 微信请求 UA | reqwest 默认 | 桌面 Chrome UA（待验证） | 降低被拦概率 |
| 浏览器显隐控件 | "隐藏/显示"按钮 | Switch | 原生控件习惯 |
| 设置页尺寸 | 500 固定 | 500 固定 | 一致 |
| 单实例 | 无 | 无 | 保持一致 |
| 配置路径 | %TEMP% | %TEMP%（同名同字段） | 兼容 |
| 注册表键 | 同左 | 同左 | 兼容 |

---

## 6. 测试策略

| 层 | 方式 | 覆盖 |
|---|---|---|
| extract | 表驱动单测 | QQ/企业微信/通用参数/fragment/双重编码/无匹配 |
| extract 微信 | testdata 里的真实 HTML 片段 + httptest | 3 种正则、base64、redirecthelpcgi 分支、超时 |
| browser | 表驱动单测 | 命令行解析（带引号/不带引号/带参数/空） |
| config | 往返测试 | 读写、缺失文件、损坏 JSON |
| view | `ui.NewTester` | 初始渲染 URL、按钮数量、点设置切换、点浏览器回调 |
| sysint | Windows 实机手测 | 枚举、注册、卸载、提权失败路径 |
| 端到端 | Windows 实机 | 设为默认浏览器 → 点 QQ 链接 → 选浏览器打开 |

---

## 7. 里程碑与验收

**M0 环境与脚手架**
- 验证 `GOTOOLCHAIN=auto` 能拉到 go1.27.3
- `go tool mygo init -template native` 生成骨架（或手工建）
- Linux 上能弹出空白 Native UI 窗口（`go tool mygo dev`）
- 验收：窗口能开能关，`go test` 通过

**M1 纯逻辑移植**（不碰 UI）
- extract / config / browser 三个包 + 单测
- 验收：`go test ./...` 全绿；用 3 条真实链接对比 Rust 版输出一致

**M2 主界面**
- 窗口参数、标题栏、URL 区、状态行、浏览器按钮、Toast、失焦关闭、动态高度
- 验收：Linux 上视觉效果与 Rust 截图对照；Windows 上启动浏览器成功

**M3 微信异步提取**
- goroutine + `win.Update`，状态机 idle/loading/ok/failed
- 验收：真实微信拦截链接能出真实地址；失败有明确文案；10s 超时

**M4 设置页 + 系统集成**
- 显隐开关持久化；注册/卸载 + 提权 + 结果提示
- 验收：Windows 设置里出现 fuckHttp 可选为默认浏览器；卸载后消失

**M5 Windows 实机 + 打包**
- 图标、版本信息、NSIS 安装包
- 验收：安装 → 设为默认 → 点击链接 → 选择浏览器 → 打开；卸载干净

**M6 收尾**
- README 更新（构建方式、Go 版说明）
- 决策：Rust 版保留 / 归档到 `legacy/` / 删除

---

## 8. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| 本机 Go 1.26.5 < 1.27.3 | 无法编译 | `GOTOOLCHAIN=auto` 自动下载；M0 首项验证，失败则手工装 1.27.3 |
| mygo v0.7.x API 仍在演进 | 编译/行为漂移 | go.mod 锁 v0.7.4；只用文档里明确列出的 API |
| Windows 上 Frameless+Transparent 渲染差异 | 圆角/阴影不一致 | M2 实机截图对照，必要时退化为不透明 + 圆角 |
| 微信页面结构再变 | 提取失败 | 保留三条正则 + 兜底分支；失败时仍可手动复制原链接 |
| 提权 API 签名手写 | 崩溃 | 改用 `arduino/go-windows-runas`（已打 tag，MIT） |
| 注册表 32/64 位视图 | 枚举不到部分浏览器 | 与原版一致（默认视图）；如出问题再加 `KEY_WOW64_64KEY` |
| 非 Windows 无法测注册表 | 回归风险 | 接口 + 桩实现，CI 跑 Linux 单测；注册表逻辑保持薄 |

---

## 9. 可选改进（不在本次范围，需单独确认）

1. 启动浏览器时保留命令里的额外参数（`%1` 替换），修复 Chrome/Edge 的 `--single-argument` 场景。
2. 配置从 `%TEMP%` 迁到 `%APPDATA%\fuckHttp`（Temp 会被清理）。
3. 记住上次选择的浏览器 / 数字键快捷选择。
4. 支持 macOS/Linux（枚举 `.desktop` / LaunchServices）。

---

## 10. 已确认的决策

1. **代码位置**：根目录即 Go 项目；Rust 版整体移到 `legacy/`（用 `git mv` 保留历史）。
2. **Rust 版去留**：归档保留，不删除。
3. **§5 的行为差异**：全部接受。

---

## 11. 实现状态

| 里程碑 | 状态 | 说明 |
|---|---|---|
| M0 环境与脚手架 | 完成 | 自动拉取 go1.27.3；mygo v0.7.4 可用；Linux 上窗口可创建 |
| M1 纯逻辑移植 | 完成 | extract / config / browser 三包，单测覆盖 |
| M2 主界面 | 完成 | 无边框透明置顶窗口、标题栏、URL 区、状态行、按钮、Toast、失焦关闭、动态高度 |
| M3 微信异步提取 | 完成 | goroutine + `Window.Update`；10s 超时；三条正则 + base64 + redirecthelpcgi 兜底 |
| M4 设置页 + 系统集成 | 代码完成 | 注册表读写、递归删除、`go-windows-runas` 提权；**待 Windows 实机验证** |
| M5 Windows 实机 + 打包 | 部分完成 | `mygo build -platform windows/amd64` 已产出 `dist/windows-amd64/fuckHttp.exe`；**安装包与默认浏览器流程待 Windows 验证** |
| M6 收尾 | 完成 | README 重写、.gitignore、Rust 归档 |
| Linux 支持 | 完成 | 按 XDG 规范实现浏览器发现与默认浏览器注册；界面修复（图标居中、空列表高度） |

### 已验证

- `go build ./...`、`go vet ./...`、`go test ./...` 全绿
- `GOOS=windows GOARCH=amd64 go build` 通过（Windows 专用代码参与编译）
- `go tool mygo build -platform windows/amd64 -skip-installer` 产出 11.7 MB 可执行文件
- 界面快照（浅色/深色/设置页）经 `ui.NewTester` 渲染确认

### 待办

- Linux 实机：在 KDE/GNOME 上验证 `xdg-settings` 与 `mimeapps.list` 的实际生效情况
- Windows 实机：浏览器枚举、注册/卸载、UAC 提权、真实链接打开
- 微信拦截链接的真实抓取验证（User-Agent 与页面结构）
- NSIS 安装包（在 Windows 上 `go tool mygo build`，或安装 NSIS 后去掉 `-skip-installer`）
