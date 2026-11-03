#![windows_subsystem = "windows"]

use eframe::{egui, NativeOptions};
use is_elevated::is_elevated;
use regex::Regex;
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;
use std::process::Command;
use winreg::enums::*;
use winreg::RegKey;
use url::Url;

// 从安全中心拦截的URL中提取真正的链接（同步版本，不处理微信）
fn extract_real_url_sync(input_url: &str) -> (String, bool) {
    // 尝试解析URL
    if let Ok(parsed_url) = Url::parse(input_url) {
        let host = parsed_url.host_str().unwrap_or("");
        
        // 微信拦截页面需要异步处理
        if host.contains("weixin110.qq.com") {
            return (input_url.to_string(), true); // 返回原URL和需要异步处理的标志
        }
        
        // 企业微信拦截页面处理
         if host.contains("open.work.weixin.qq.com") {
             let query_pairs: std::collections::HashMap<_, _> = parsed_url.query_pairs().collect();
             if let Some(uri) = query_pairs.get("uri") {
                 let decoded_uri = urlencoding::decode(uri).unwrap_or_else(|_| uri.clone());
                 if !decoded_uri.starts_with("http://") && !decoded_uri.starts_with("https://") {
                     return (format!("https://{}", decoded_uri), false);
                 }
                 return (decoded_uri.to_string(), false);
             }
         }
         
         // QQ电脑版拦截页面处理
         if host.contains("c.pc.qq.com") {
             let query_pairs: std::collections::HashMap<_, _> = parsed_url.query_pairs().collect();
             if let Some(url_param) = query_pairs.get("url") {
                 let decoded_url = urlencoding::decode(url_param).unwrap_or_else(|_| url_param.clone());
                 if decoded_url.starts_with("http://") || decoded_url.starts_with("https://") {
                     return (decoded_url.to_string(), false);
                 }
             }
         }
        
        // 通用URL参数提取
        let query_pairs: std::collections::HashMap<_, _> = parsed_url.query_pairs().collect();
        let url_params = ["url", "link", "target", "redirect", "goto", "u", "q"];
        
        for param in &url_params {
            if let Some(extracted_url) = query_pairs.get(*param) {
                let decoded_url = urlencoding::decode(extracted_url).unwrap_or_else(|_| extracted_url.clone());
                if decoded_url.starts_with("http://") || decoded_url.starts_with("https://") {
                     return (decoded_url.to_string(), false);
                 }
             }
         }
         
         // 检查fragment部分
         if let Some(fragment) = parsed_url.fragment() {
             if fragment.starts_with("http://") || fragment.starts_with("https://") {
                 return (fragment.to_string(), false);
             }
         }
     }
     
     // 如果无法提取，返回原始URL
     (input_url.to_string(), false)
 }

// 解码HTML实体
fn decode_html_entities(s: &str) -> String {
    s.replace("&#x2f;", "/")
        .replace("&#x3a;", ":")
        .replace("&#x3d;", "=")
        .replace("&#x3f;", "?")
        .replace("&#x26;", "&")
        .replace("&amp;", "&")
        .replace("&lt;", "<")
        .replace("&gt;", ">")
        .replace("&quot;", "\"")
}

// 从微信拦截页面提取真实链接
fn extract_from_wechat_page(wechat_url: &str) -> Option<String> {
    // 使用阻塞式HTTP客户端访问微信页面
    if let Ok(response) = reqwest::blocking::get(wechat_url) {
        if let Ok(html) = response.text() {
            // 方法1: 尝试从"继续访问"按钮提取（type: "gray" 的情况）
            if let Ok(regex) = Regex::new(r#""name"\s*:\s*"继续访问",\s*"url"\s*:\s*"([^"]+)""#) {
                if let Some(captures) = regex.captures(&html) {
                    if let Some(url_match) = captures.get(1) {
                        let decoded = decode_html_entities(url_match.as_str());
                        if decoded.starts_with("http://") || decoded.starts_with("https://") {
                            return Some(decoded);
                        }
                    }
                }
            }
            
            // 方法2: 尝试从desc字段提取（type: "empty" 的情况）
            if let Ok(regex) = Regex::new(r#""desc"\s*:\s*"([^"]+)""#) {
                if let Some(captures) = regex.captures(&html) {
                    if let Some(desc_match) = captures.get(1) {
                        let decoded = decode_html_entities(desc_match.as_str());
                        if decoded.starts_with("http://") || decoded.starts_with("https://") {
                            return Some(decoded);
                        }
                    }
                }
            }
            
            // 方法3: 从"申请恢复访问"按钮获取并解码base64（type: "block" 的情况）
            if let Ok(regex) = Regex::new(r#""name"\s*:\s*"申请恢复访问",\s*"url"\s*:\s*"([^"]+)""#) {
                if let Some(captures) = regex.captures(&html) {
                    if let Some(url_match) = captures.get(1) {
                        // 先解码HTML实体
                        let restore_url = decode_html_entities(url_match.as_str());
                        
                        // 先尝试直接从申请恢复访问链接的url参数解码（新格式）
                        if let Ok(parsed_url) = url::Url::parse(&restore_url) {
                            let query_pairs: std::collections::HashMap<_, _> = parsed_url.query_pairs().collect();
                            if let Some(encoded_url) = query_pairs.get("url") {
                                use base64::Engine;
                                if let Ok(decoded_bytes) = base64::engine::general_purpose::STANDARD.decode(encoded_url.as_ref()) {
                                    if let Ok(final_url) = String::from_utf8(decoded_bytes) {
                                        if final_url.starts_with("http://") || final_url.starts_with("https://") {
                                            return Some(final_url);
                                        }
                                    }
                                }
                            }
                        }
                        
                        // 如果直接解码失败，尝试二次访问（旧格式）
                        // 将newreadtemplate替换为redirecthelpcgi
                        let redirect_url = restore_url.replace("newreadtemplate", "redirecthelpcgi");
                        
                        // 访问redirecthelpcgi链接获取JSON响应
                        if let Ok(json_response) = reqwest::blocking::get(&redirect_url) {
                            if let Ok(json_text) = json_response.text() {
                                // 解析JSON并提取申请恢复按钮的URL
                                if let Ok(json_value) = serde_json::from_str::<serde_json::Value>(&json_text) {
                                    if let Some(btns) = json_value["btns"].as_array() {
                                        for btn in btns {
                                            if let Some(name) = btn["name"].as_str() {
                                                if name == "申请恢复" {
                                                    if let Some(btn_url) = btn["url"].as_str() {
                                                        // 从URL中提取base64编码的参数
                                                        if let Ok(parsed_url) = url::Url::parse(btn_url) {
                                                            let query_pairs: std::collections::HashMap<_, _> = parsed_url.query_pairs().collect();
                                                            if let Some(encoded_url) = query_pairs.get("url") {
                                                                use base64::Engine;
                                                                if let Ok(decoded_bytes) = base64::engine::general_purpose::STANDARD.decode(encoded_url.as_ref()) {
                                                                    if let Ok(final_url) = String::from_utf8(decoded_bytes) {
                                                                        if final_url.starts_with("http://") || final_url.starts_with("https://") {
                                                                            return Some(final_url);
                                                                        }
                                                                    }
                                                                }
                                                            }
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
    None
}

#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
struct Browser {
    name: String,
    command: String,
    #[serde(default)]
    hidden: bool,
}

#[derive(Serialize, Deserialize, Default)]
struct Config {
    hidden_browsers: Vec<String>,
}

fn get_config_path() -> PathBuf {
    let mut path = std::env::temp_dir();
    path.push("browser_selector_config.json");
    path
}

fn load_config() -> Config {
    fs::read_to_string(get_config_path())
        .ok()
        .and_then(|content| serde_json::from_str(&content).ok())
        .unwrap_or_default()
}

fn save_config(config: &Config) {
    if let Ok(content) = serde_json::to_string_pretty(config) {
        fs::write(get_config_path(), content).ok();
    }
}

fn get_browsers_from_hive(hive: &RegKey, browsers: &mut Vec<Browser>) {
    if let Ok(key) = hive.open_subkey("SOFTWARE\\Clients\\StartMenuInternet") {
        for subkey_name in key.enum_keys().filter_map(Result::ok) {
            if let Ok(subkey) = key.open_subkey(&subkey_name) {
                if let Ok(name) = subkey.get_value::<String, _>("") {
                    if let Ok(command_key) = subkey.open_subkey("shell\\open\\command") {
                        if let Ok(command) = command_key.get_value::<String, _>("") {
                            if !browsers.iter().any(|b| b.name == name) {
                                browsers.push(Browser {
                                    name,
                                    command,
                                    hidden: false,
                                });
                            }
                        }
                    }
                }
            }
        }
    }
}

fn get_installed_browsers() -> Vec<Browser> {
    let mut browsers = Vec::new();
    let config = load_config();

    let hklm = RegKey::predef(HKEY_LOCAL_MACHINE);
    get_browsers_from_hive(&hklm, &mut browsers);

    let hkcu = RegKey::predef(HKEY_CURRENT_USER);
    get_browsers_from_hive(&hkcu, &mut browsers);

    for browser in &mut browsers {
        if config.hidden_browsers.contains(&browser.name) {
            browser.hidden = true;
        }
    }

    browsers
}

#[derive(Debug, Clone)]
enum UrlExtractionState {
    Pending,
    Loading,
    Success,
    Failed(String),
}

struct BrowserSelectorApp {
    browsers: Vec<Browser>,
    url_to_open: String,
    original_url: String,
    show_settings: bool,
    message: Option<String>,
    last_window_height: f32,
    last_click_time: std::time::Instant,
    toast_message: Option<(String, std::time::Instant)>,
    first_frame: bool,
    url_extraction_state: UrlExtractionState,
    wechat_extraction_handle: Option<std::thread::JoinHandle<Option<String>>>,
}

impl BrowserSelectorApp {
    fn new(cc: &eframe::CreationContext<'_>, url_to_open: String, browsers: Vec<Browser>) -> Self {
        let mut fonts = egui::FontDefinitions::default();

        if let Ok(font_data) = std::fs::read("C:\\Windows\\Fonts\\msyh.ttc") {
            fonts
                .font_data
                .insert("my_font".to_owned(), egui::FontData::from_owned(font_data));
            fonts
                .families
                .entry(egui::FontFamily::Proportional)
                .or_default()
                .insert(0, "my_font".to_owned());
        } else if let Ok(font_data) = std::fs::read("C:\\Windows\\Fonts\\simhei.ttf") {
            fonts
                .font_data
                .insert("my_font".to_owned(), egui::FontData::from_owned(font_data));
            fonts
                .families
                .entry(egui::FontFamily::Proportional)
                .or_default()
                .insert(0, "my_font".to_owned());
        }

        cc.egui_ctx.set_fonts(fonts);

        // 提取真实URL
        let original_url = url_to_open.clone();
        let (extracted_url, needs_async) = extract_real_url_sync(&url_to_open);
        
        let url_extraction_state = if needs_async {
            UrlExtractionState::Pending
        } else {
            UrlExtractionState::Success
        };

        Self {
            browsers,
            url_to_open: extracted_url,
            original_url,
            show_settings: false,
            message: None,
            last_window_height: 0.0,
            last_click_time: std::time::Instant::now(),
            toast_message: None,
            first_frame: true,
            url_extraction_state,
            wechat_extraction_handle: None,
        }
    }
}

fn register_app() -> std::io::Result<()> {
    let hkcu = RegKey::predef(HKEY_CURRENT_USER);
    let path = "Software\\Classes\\fuckHttp";
    let (key, _) = hkcu.create_subkey(&path)?;

    key.set_value("", &"URL:fuckHttp Protocol")?;
    key.set_value("URL Protocol", &"")?;

    let (icon_key, _) = key.create_subkey("DefaultIcon")?;
    let exe_path = std::env::current_exe()?;
    icon_key.set_value("", &format!("\"{}\",0", exe_path.to_str().unwrap()))?;

    let (command_key, _) = key.create_subkey("shell\\open\\command")?;
    command_key.set_value(
        "",
        &format!("\"{}\" \"%1\"", exe_path.to_str().unwrap()),
    )?;

    let hklm = RegKey::predef(HKEY_LOCAL_MACHINE);
    let capabilities_path = "Software\\fuckHttp\\Capabilities";
    let (capabilities_key, _) = hklm.create_subkey(capabilities_path)?;
    capabilities_key.set_value("ApplicationName", &"fuckHttp")?;
    capabilities_key.set_value(
        "ApplicationIcon",
        &format!("\"{}\",0", exe_path.to_str().unwrap()),
    )?;
    capabilities_key.set_value("ApplicationDescription", &"A custom browser selector.")?;

    let (url_assoc_key, _) = capabilities_key.create_subkey("URLAssociations")?;
    url_assoc_key.set_value("http", &"fuckHttp")?;
    url_assoc_key.set_value("https", &"fuckHttp")?;

    let registered_apps_path = "Software\\RegisteredApplications";
    let registered_apps_key = hklm.open_subkey_with_flags(registered_apps_path, KEY_WRITE)?;
    registered_apps_key.set_value(
        "fuckHttp",
        &"Software\\fuckHttp\\Capabilities".to_string(),
    )?;

    Ok(())
}

fn unregister_app() -> std::io::Result<()> {
    let hkcu = RegKey::predef(HKEY_CURRENT_USER);
    hkcu.delete_subkey_all("Software\\Classes\\fuckHttp")?;

    let hklm = RegKey::predef(HKEY_LOCAL_MACHINE);
    hklm.delete_subkey_all("Software\\fuckHttp")?;

    let registered_apps_key =
        hklm.open_subkey_with_flags("Software\\RegisteredApplications", KEY_WRITE)?;
    registered_apps_key.delete_value("fuckHttp")?;

    Ok(())
}

impl eframe::App for BrowserSelectorApp {
    fn update(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {
        // 处理异步URL提取
        match &self.url_extraction_state {
            UrlExtractionState::Pending => {
                // 启动异步微信链接提取
                let url = self.original_url.clone();
                let handle = std::thread::spawn(move || {
                    extract_from_wechat_page(&url)
                });
                self.wechat_extraction_handle = Some(handle);
                self.url_extraction_state = UrlExtractionState::Loading;
                ctx.request_repaint();
            }
            UrlExtractionState::Loading => {
                // 检查异步任务是否完成
                if let Some(handle) = self.wechat_extraction_handle.take() {
                    if handle.is_finished() {
                        match handle.join() {
                            Ok(Some(real_url)) => {
                                self.url_to_open = real_url;
                                self.url_extraction_state = UrlExtractionState::Success;
                            }
                            Ok(None) => {
                                self.url_extraction_state = UrlExtractionState::Failed("无法从微信页面提取链接".to_string());
                            }
                            Err(_) => {
                                self.url_extraction_state = UrlExtractionState::Failed("网络请求失败".to_string());
                            }
                        }
                        ctx.request_repaint();
                    } else {
                        // 任务还在进行中，放回handle
                        self.wechat_extraction_handle = Some(handle);
                        ctx.request_repaint_after(std::time::Duration::from_millis(100));
                    }
                }
            }
            _ => {}
        }
        
        // 计算当前窗口高度（根据实际浏览器数量）
        let mut window_height = 20.0; // 基础边距
        
        // URL提取状态提示（所有状态都需要预留空间）
        match &self.url_extraction_state {
            UrlExtractionState::Loading | UrlExtractionState::Success | UrlExtractionState::Failed(_) => {
                window_height += 20.0;
            }
            UrlExtractionState::Pending => {
                // 对于非异步处理的URL，如果已提取则显示
                if self.original_url != self.url_to_open {
                    window_height += 20.0;
                }
            }
        }
        
        // URL滚动框：固定高度
        window_height += 60.0;
        
        // 分隔线
        window_height += 20.0;
        
        // 浏览器选项高度（根据实际可见浏览器数量）
        let visible_browsers_count = self.browsers.iter().filter(|b| !b.hidden).count();
        if visible_browsers_count > 0 {
            window_height += visible_browsers_count as f32 * 50.0;
            window_height += 20.0;
        }
        
        // 底部边距
        window_height += 25.0;
        
        // 设置界面固定高度
        if self.show_settings {
            window_height = 500.0;
        }
        
        // 在第一帧或高度变化时立即设置窗口尺寸
        if self.first_frame || (self.last_window_height - window_height).abs() > 1.0 {
            if self.first_frame {
                self.first_frame = false;
            }
            self.last_window_height = window_height;
            ctx.send_viewport_cmd(egui::ViewportCommand::InnerSize(egui::vec2(400.0, window_height)));
            ctx.request_repaint();
        }

        let mut style = (*ctx.style()).clone();
        match dark_light::detect() {
            dark_light::Mode::Dark => {
                style.visuals = egui::Visuals::dark();
            }
            dark_light::Mode::Light => {
                style.visuals = egui::Visuals::light();
            }
            dark_light::Mode::Default => {
                style.visuals = egui::Visuals::light();
            }
        }

        style.visuals.widgets.inactive.rounding = egui::Rounding::from(5.0);
        style.visuals.widgets.hovered.rounding = egui::Rounding::from(5.0);
        style.visuals.widgets.active.rounding = egui::Rounding::from(5.0);
        style.visuals.window_rounding = egui::Rounding::from(8.0);
        ctx.set_style(style);

        egui::TopBottomPanel::top("title_bar").show(ctx, |ui| {
            ui.horizontal(|ui| {
                ui.heading("fuckHttp");
                ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
                    ui.style_mut().spacing.button_padding = egui::vec2(4.0, 2.0);
                    if ui.add(egui::Button::new("❌").small()).clicked() {
                        ctx.send_viewport_cmd(egui::ViewportCommand::Close);
                    }
                    if ui.add(egui::Button::new("🔧").small()).clicked() {
                        self.show_settings = !self.show_settings;
                        self.message = None; // Clear message when toggling settings
                    }
                    if ui.add(egui::Button::new("📋").small()).on_hover_text("复制链接").clicked() {
                        ui.output_mut(|o| o.copied_text = self.url_to_open.clone());
                        self.toast_message = Some(("链接已复制到剪贴板".to_string(), std::time::Instant::now()));
                    }
                });
            });
        });

        if self.show_settings {
            egui::CentralPanel::default().show(ctx, |ui| {
                ui.heading("设置");
                ui.add_space(10.0);

                ui.separator();
                ui.heading("浏览器列表");
                let mut config_changed = false;
                for browser in &mut self.browsers {
                    ui.horizontal(|ui| {
                        ui.label(&browser.name);
                        let button_text = if browser.hidden { "显示" } else { "隐藏" };
                        if ui.button(button_text).clicked() {
                            browser.hidden = !browser.hidden;
                            config_changed = true;
                        }
                    });
                }

                if config_changed {
                    let hidden_browsers = self
                        .browsers
                        .iter()
                        .filter(|b| b.hidden)
                        .map(|b| b.name.clone())
                        .collect();
                    save_config(&Config { hidden_browsers });
                }

                ui.separator();

                ui.add_space(10.0);

                if ui.button("注册到系统").clicked() {
                    if !is_elevated() {
                        let exe = std::env::current_exe().unwrap();
                        match runas::Command::new(exe).arg("--register").status() {
                            Ok(status) if status.success() => {
                                self.message = Some("注册成功!".to_string());
                            }
                            _ => {
                                self.message = Some("注册失败 (需要管理员权限).".to_string());
                            }
                        }
                    } else {
                        match register_app() {
                            Ok(_) => self.message = Some("注册成功!".to_string()),
                            Err(e) => {
                                self.message = Some(format!("注册失败: {}", e));
                            }
                        }
                    }
                }
                ui.add_space(5.0);
                if ui.button("从系统卸载").clicked() {
                    if !is_elevated() {
                        let exe = std::env::current_exe().unwrap();
                        match runas::Command::new(exe).arg("--unregister").status() {
                            Ok(status) if status.success() => {
                                self.message = Some("卸载成功!".to_string());
                            }
                            _ => {
                                self.message = Some("卸载失败 (需要管理员权限).".to_string());
                            }
                        }
                    } else {
                        match unregister_app() {
                            Ok(_) => self.message = Some("卸载成功!".to_string()),
                            Err(e) => {
                                self.message = Some(format!("卸载失败: {}", e));
                            }
                        }
                    }
                }
                if let Some(msg) = &self.message {
                    ui.add_space(10.0);
                    ui.label(msg);
                }
            });
        } else {
            egui::CentralPanel::default()
                 .frame(
                     egui::Frame::none()
                         .inner_margin(egui::Margin::same(10.0))
                         .fill(ctx.style().visuals.window_fill()),
                 )
                 .show(ctx, |ui| {
                    ui.add_space(4.0);
                    
                    // 显示URL提取状态
                    match &self.url_extraction_state {
                        UrlExtractionState::Loading => {
                            ui.horizontal(|ui| {
                                ui.add(egui::Label::new(egui::RichText::new("⏳ 正在获取真实链接...").color(egui::Color32::from_rgb(255, 165, 0)).size(12.0)));
                            });
                            ui.add_space(2.0);
                        }
                        UrlExtractionState::Success => {
                            if self.original_url != self.url_to_open {
                                ui.horizontal(|ui| {
                                    ui.add(egui::Label::new(egui::RichText::new("🔓 已提取真实链接").color(egui::Color32::from_rgb(0, 150, 0)).size(12.0)));
                                });
                                ui.add_space(2.0);
                            }
                        }
                        UrlExtractionState::Failed(error) => {
                            ui.horizontal(|ui| {
                                ui.add(egui::Label::new(egui::RichText::new(&format!("❌ 提取失败: {}", error)).color(egui::Color32::from_rgb(255, 0, 0)).size(12.0)));
                            });
                            ui.add_space(2.0);
                        }
                        _ => {}
                    }
                    
                    // URL显示区域：强制固定60px高度容器
                    let url_rect = ui.allocate_response(egui::vec2(ui.available_width(), 60.0), egui::Sense::hover());
                    ui.allocate_ui_at_rect(url_rect.rect, |ui| {
                        ui.set_clip_rect(url_rect.rect); // 强制裁剪超出部分
                        egui::ScrollArea::vertical()
                            .max_height(60.0)
                            .auto_shrink([false, false])
                            .show(ui, |ui| {
                                ui.add(egui::Label::new(egui::RichText::new(&self.url_to_open).size(14.0)).wrap(true));
                            });
                    });
                    ui.add_space(8.0);

                    ui.separator();

                    ui.add_space(10.0);
                    let button_width = ui.available_width() - 20.0;
                    ui.vertical_centered(|ui| {
                        let visible_browsers: Vec<_> = self.browsers.iter().filter(|b| !b.hidden).collect();
                        for (index, browser) in visible_browsers.iter().enumerate() {
                            let button = egui::Button::new(&browser.name)
                                .min_size(egui::vec2(button_width, 40.0));
                            if ui.add(button).clicked() {
                                // 更安全的命令解析方式
                                let command = browser.command.trim();
                                let executable = if command.starts_with('"') {
                                    // 处理带引号的路径
                                    if let Some(end_quote) = command[1..].find('"') {
                                        &command[1..end_quote + 1]
                                    } else {
                                        command
                                    }
                                } else {
                                    // 处理不带引号的路径，取第一个空格前的部分
                                    command.split_whitespace().next().unwrap_or(command)
                                };

                                if !executable.is_empty() {
                                    if let Err(e) = Command::new(executable)
                                        .arg(&self.url_to_open)
                                        .spawn() {
                                        eprintln!("启动浏览器失败: {}", e);
                                    }
                                }
                                ctx.send_viewport_cmd(egui::ViewportCommand::Close);
                            }
                            // 只在不是最后一个按钮时添加间距
                            if index < visible_browsers.len() - 1 {
                                ui.add_space(5.0);
                            }
                        }
                        // 在所有按钮后添加底部边距
                        ui.add_space(5.0);
                    });
                });
        }

        // 点击外部区域关闭窗口，但避免在文本选择时误触发
        ctx.input(|i| {
            if i.pointer.any_pressed() {
                self.last_click_time = std::time::Instant::now();
            }
        });
        
        // 只有在点击（而非拖拽）且不在UI区域内时才关闭窗口
        if !ctx.is_pointer_over_area() && 
           ctx.input(|i| i.pointer.any_pressed()) && 
           !ctx.input(|i| i.pointer.is_decidedly_dragging()) {
            // 添加短暂延迟，避免误触发
            if self.last_click_time.elapsed().as_millis() > 100 {
                 ctx.send_viewport_cmd(egui::ViewportCommand::Close);
             }
         }

         // 显示toast消息
          if let Some((message, time)) = &self.toast_message {
              if time.elapsed().as_secs() < 3 {
                  egui::Window::new("toast")
                      .title_bar(false)
                      .resizable(false)
                      .collapsible(false)
                      .anchor(egui::Align2::CENTER_BOTTOM, egui::vec2(0.0, -20.0))
                      .frame(egui::Frame::popup(&ctx.style()).fill(egui::Color32::from_rgba_premultiplied(0, 0, 0, 200)))
                      .show(ctx, |ui| {
                          ui.add(egui::Label::new(egui::RichText::new(message).color(egui::Color32::WHITE).size(14.0)));
                      });
              } else {
                  self.toast_message = None;
              }
          }
    }
}

fn load_icon(path: &str) -> Option<egui::IconData> {
    let (icon_rgba, icon_width, icon_height) = {
        let image = image::open(path).ok()?.to_rgba8();
        let (width, height) = image.dimensions();
        let rgba = image.into_raw();
        (rgba, width, height)
    };

    Some(egui::IconData {
        rgba: icon_rgba,
        width: icon_width,
        height: icon_height,
    })
}

fn main() -> Result<(), eframe::Error> {
    let args: Vec<String> = std::env::args().collect();

    if args.contains(&"--register".to_string()) {
        if is_elevated() {
            match register_app() {
                Ok(_) => std::process::exit(0),
                Err(_) => std::process::exit(1),
            }
        }
    } else if args.contains(&"--unregister".to_string()) {
        if is_elevated() {
            match unregister_app() {
                Ok(_) => std::process::exit(0),
                Err(_) => std::process::exit(1),
            }
        }
    }

    let url_to_open = if args.len() > 1 && !args[1].starts_with("--") {
        args[1].clone()
    } else {
        "https://www.google.com".to_string()
    };

    let all_browsers = get_installed_browsers();
    
    // 计算初始窗口高度，避免越界
    let mut initial_height = 20.0; // 基础边距
    
    // URL提取状态提示（假设可能有）
    let (extracted_url, _) = extract_real_url_sync(&url_to_open);
    if extracted_url != url_to_open {
        initial_height += 20.0;
    }
    
    // URL滚动框：固定高度
    initial_height += 60.0;
    
    // 分隔线
    initial_height += 20.0;
    
    // 浏览器选项高度
    let visible_browsers_count = all_browsers.iter().filter(|b| !b.hidden).count();
    if visible_browsers_count > 0 {
        initial_height += visible_browsers_count as f32 * 50.0;
        initial_height += 20.0; // 额外底部边距
    }
    
    // 底部边距
    initial_height += 25.0;

    let mut viewport = egui::ViewportBuilder::default()
        .with_inner_size([400.0, initial_height]) // 使用计算的初始高度
        .with_decorations(false)
        .with_always_on_top()
        .with_window_level(egui::WindowLevel::AlwaysOnTop)
        .with_resizable(false) // 通过代码控制大小，不允许手动调整
        .with_transparent(true);

    if let Some(icon) = load_icon("./icon.ico") {
        viewport = viewport.with_icon(icon);
    }

    let options = NativeOptions {
        centered: true,
        follow_system_theme: false,
        viewport,
        ..Default::default()
    };

    eframe::run_native(
        "fuckHttp",
        options,
        Box::new(move |cc| Box::new(BrowserSelectorApp::new(cc, url_to_open, all_browsers))),
    )
}
