use std::sync::Mutex;
use tauri::{AppHandle, Manager, State, WebviewUrl, WebviewWindow, WebviewWindowBuilder};

#[derive(Default)]
struct DesktopSession(Mutex<Option<String>>);

#[tauri::command]
fn desktop_session(state: State<'_, DesktopSession>) -> Result<Option<String>, String> {
    state
        .0
        .lock()
        .map(|token| token.clone())
        .map_err(|_| "无法读取桌面会话".into())
}

#[tauri::command]
fn set_desktop_session(
    state: State<'_, DesktopSession>,
    token: Option<String>,
) -> Result<(), String> {
    let mut session = state.0.lock().map_err(|_| "无法更新桌面会话")?;
    *session = token;
    Ok(())
}

#[tauri::command]
async fn open_admin(app: AppHandle, caller: WebviewWindow) -> Result<(), String> {
    if caller.label() != "main" || desktop_session(app.state())?.is_none() {
        return Err("请先登录工作台".into());
    }
    if let Some(window) = app.get_webview_window("admin") {
        window.unminimize().map_err(|err| err.to_string())?;
        return window.set_focus().map_err(|err| err.to_string());
    }
    WebviewWindowBuilder::new(
        &app,
        "admin",
        WebviewUrl::App("index.html#/admin/accounts".into()),
    )
    .title("中网联合 · 管理面板")
    .inner_size(1160.0, 760.0)
    .min_inner_size(760.0, 500.0)
    .decorations(false)
    .build()
    .map_err(|err| err.to_string())?;
    Ok(())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .manage(DesktopSession::default())
        .invoke_handler(tauri::generate_handler![
            desktop_session,
            set_desktop_session,
            open_admin
        ])
        .run(tauri::generate_context!())
        .expect("无法启动中网联合 OA / CRM 桌面端");
}
