mod credentials;

use credentials::DesktopSession;
use tauri::{AppHandle, Manager, State, WebviewUrl, WebviewWindow, WebviewWindowBuilder};

#[tauri::command]
fn desktop_session(
    state: State<'_, DesktopSession>,
    scope: String,
) -> Result<Option<String>, String> {
    state.read(&scope)
}

#[tauri::command]
fn set_desktop_session(
    state: State<'_, DesktopSession>,
    scope: String,
    token: String,
) -> Result<(), String> {
    state.store(&scope, &token)
}

#[tauri::command]
fn clear_desktop_session(
    state: State<'_, DesktopSession>,
    scope: String,
    token: String,
) -> Result<bool, String> {
    state.clear_if_current(&scope, &token)
}

#[tauri::command]
async fn open_admin(app: AppHandle, caller: WebviewWindow, scope: String) -> Result<(), String> {
    if caller.label() != "main" || app.state::<DesktopSession>().read(&scope)?.is_none() {
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
            clear_desktop_session,
            open_admin
        ])
        .run(tauri::generate_context!())
        .expect("无法启动中网联合 OA / CRM 桌面端");
}
