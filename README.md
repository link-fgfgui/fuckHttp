# fuckHttp 浏览器选择器

一个桌面程序，用于拦截被 App 劫持的 URL，提取真实链接，并让你选择用哪个浏览器打开。支持 Windows 与 Linux。

故事背景：https://linux.do/t/topic/914393

本仓库现在有两个实现：

- **根目录**：Go + [MyGo](https://github.com/egoist/mygo) 重写版（Native UI，纯 Go，无 webview）
- **`legacy/`**：最初的 Rust + egui 版本，保留作参考

## 功能特性

- 拦截来自 QQ、微信、企业微信安全页面的 URL
- 从重定向页面提取真实 URL（支持双重编码、base64、微信页面异步抓取）
- 图形界面选择浏览器
- 支持隐藏不常用的浏览器
- 复制链接、系统注册/卸载
- 注册为默认浏览器候选（Windows 注册表 / Linux 桌面条目 + mimeapps.list）

## 环境要求

- Go **1.27.3** 或更高（`GOTOOLCHAIN=auto` 会自动下载）
- 构建 Windows 安装包需要 NSIS（`mygo build` 会自动下载）
- 在 Linux 上开发界面需要 GTK 3

## 开发与构建

```sh
# 开发模式（改代码自动重启）
go tool mygo dev

# 构建 Windows 版
go tool mygo build -platform windows/amd64

# 构建 Linux 版
go tool mygo build -platform linux/amd64

# 在 Linux 上构建时跳过 NSIS 安装包
go tool mygo build -platform windows/amd64 -skip-installer

# 测试
go test ./...
```

产物在 `dist/<os>-<arch>/`。

## 使用

```
fuckHttp.exe "https://example.com"
```

不带参数时默认打开 `https://www.google.com`。

### 支持的 URL 模式

- QQ 电脑版：`https://c.pc.qq.com/ios.html?level=14&url=*`
- 微信：`https://weixin110.qq.com/security/readtemplate?*`
- 企业微信：`https://open.work.weixin.qq.com/wwopen/mpnews?*`
- 通用重定向参数：`url`、`link`、`target`、`redirect`、`goto`、`u`、`q`
- URL fragment 中的完整链接

## 配置

配置保存在：

```
%TEMP%\browser_selector_config.json
```

内容为被隐藏的浏览器名称，与 Rust 版兼容：

```json
{
  "hidden_browsers": ["Firefox"]
}
```

## 系统集成

设置页可以把程序注册为 HTTP/HTTPS 的默认浏览器候选。

### Windows

- `HKCU\Software\Classes\fuckHttp`：URL 协议处理器
- `HKLM\Software\fuckHttp\Capabilities`：应用能力声明
- `HKLM\Software\RegisteredApplications`：注册到"默认应用"

注册需要管理员权限，程序会通过 UAC 重新启动自己完成写入。注册后到 Windows 设置 → 默认应用里选择 fuckHttp。

### Linux

按 XDG 规范实现：

- 写入 `$XDG_DATA_HOME/applications/fuckhttp.desktop`（Desktop Entry 规范，`Exec` 用 `%u` 字段码）
- 在 `$XDG_CONFIG_HOME/mimeapps.list` 的 `[Default Applications]` 中登记
  `x-scheme-handler/http`、`x-scheme-handler/https`、`text/html`
- 尽力调用 `update-desktop-database` 与 `xdg-settings set default-web-browser`

不需要 root。卸载时删除桌面条目并从 mimeapps.list 中移除自己的名字。

### 浏览器发现

- Windows：读 `SOFTWARE\Clients\StartMenuInternet`（HKLM + HKCU）
- Linux：按 `$XDG_DATA_HOME`、`$XDG_DATA_DIRS` 的优先级扫描
  `applications/*.desktop`，取 `Categories` 含 `WebBrowser` 或
  `MimeType` 含 `x-scheme-handler/http(s)` 的条目，启动时按 Desktop Entry
  规范展开 `Exec` 的字段码

## 项目结构

```
main.go                 入口：参数、注册模式、窗口
mygo.json               应用名、版本、图标
resources/icon.png      应用图标
internal/extract/       URL 提取（QQ / 企业微信 / 微信 / 通用）
internal/browser/       浏览器枚举与启动
internal/config/        隐藏浏览器配置
internal/sysint/        注册表注册与提权
internal/view/          界面状态与视图
legacy/                 Rust 版源码
```

## 与 Rust 版的差异

- 窗口在失去焦点时关闭（Rust 版是点击窗口内空白处）
- 中文与 emoji 由系统文本引擎处理，不再手工加载字体
- 微信页面请求有 10 秒超时，并带桌面浏览器 User-Agent
- 浏览器显隐使用开关控件
