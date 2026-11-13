use sha2::{Digest, Sha256};
#[cfg(not(windows))]
use std::collections::HashMap;
use std::sync::Mutex;

#[derive(Default)]
pub struct DesktopSession(Mutex<Credentials>);

impl DesktopSession {
    pub fn read(&self, scope: &str) -> Result<Option<String>, String> {
        let target = target_name(scope)?;
        self.0.lock().map_err(|_| "无法读取桌面会话")?.read(&target)
    }

    pub fn store(&self, scope: &str, token: &str) -> Result<(), String> {
        let target = target_name(scope)?;
        if !valid_token(token) {
            return Err("桌面会话令牌格式无效".into());
        }
        self.0
            .lock()
            .map_err(|_| "无法更新桌面会话")?
            .store(&target, token)
    }

    pub fn clear_if_current(&self, scope: &str, token: &str) -> Result<bool, String> {
        let target = target_name(scope)?;
        let mut credentials = self.0.lock().map_err(|_| "无法更新桌面会话")?;
        match credentials.read(&target)? {
            Some(current) if current != token => return Ok(false),
            Some(_) => credentials.remove(&target)?,
            None => {}
        }
        Ok(true)
    }
}

fn target_name(scope: &str) -> Result<String, String> {
    if scope.is_empty() || scope.len() > 2048 || scope.chars().any(char::is_control) {
        return Err("桌面会话的服务地址无效".into());
    }
    // Windows 凭据名称不区分大小写，散列可区分 API 地址中大小写不同的路径。
    Ok(format!("cn.zwlh.zwoa/session/{:x}", Sha256::digest(scope)))
}

fn valid_token(token: &str) -> bool {
    token.len() == 43
        && token
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || matches!(byte, b'-' | b'_'))
}

#[cfg(windows)]
#[derive(Default)]
struct Credentials;

#[cfg(windows)]
impl Credentials {
    fn read(&self, target: &str) -> Result<Option<String>, String> {
        use std::{ptr, slice, str};
        use windows_sys::Win32::{
            Foundation::ERROR_NOT_FOUND,
            Security::Credentials::{CredFree, CredReadW, CREDENTIALW, CRED_TYPE_GENERIC},
        };

        let target = wide_target(target);
        let mut credential = ptr::null_mut::<CREDENTIALW>();
        // CredReadW 返回的缓冲区由 CredFree 释放，令牌先复制到 Rust 字符串。
        unsafe {
            if CredReadW(target.as_ptr(), CRED_TYPE_GENERIC, 0, &mut credential) == 0 {
                let error = std::io::Error::last_os_error();
                return if error.raw_os_error() == Some(ERROR_NOT_FOUND as i32) {
                    Ok(None)
                } else {
                    Err(format!("无法读取 Windows 会话凭据：{error}"))
                };
            }
            let entry = &*credential;
            let token = if entry.CredentialBlobSize == 43 && !entry.CredentialBlob.is_null() {
                str::from_utf8(slice::from_raw_parts(entry.CredentialBlob, 43))
                    .ok()
                    .filter(|token| valid_token(token))
                    .map(str::to_owned)
            } else {
                None
            };
            CredFree(credential.cast());
            Ok(token)
        }
    }

    fn store(&mut self, target: &str, token: &str) -> Result<(), String> {
        use windows_sys::Win32::Security::Credentials::{
            CredWriteW, CREDENTIALW, CRED_PERSIST_LOCAL_MACHINE, CRED_TYPE_GENERIC,
        };

        let mut target = wide_target(target);
        let mut bytes = token.as_bytes().to_vec();
        let entry = CREDENTIALW {
            Type: CRED_TYPE_GENERIC,
            TargetName: target.as_mut_ptr(),
            CredentialBlobSize: bytes.len() as u32,
            CredentialBlob: bytes.as_mut_ptr(),
            Persist: CRED_PERSIST_LOCAL_MACHINE,
            ..Default::default()
        };
        // 两个缓冲区在调用返回前保持有效。
        if unsafe { CredWriteW(&entry, 0) } == 0 {
            return Err(format!(
                "无法保存 Windows 会话凭据：{}",
                std::io::Error::last_os_error()
            ));
        }
        Ok(())
    }

    fn remove(&mut self, target: &str) -> Result<(), String> {
        use windows_sys::Win32::{
            Foundation::ERROR_NOT_FOUND,
            Security::Credentials::{CredDeleteW, CRED_TYPE_GENERIC},
        };

        let target = wide_target(target);
        if unsafe { CredDeleteW(target.as_ptr(), CRED_TYPE_GENERIC, 0) } == 0 {
            let error = std::io::Error::last_os_error();
            if error.raw_os_error() != Some(ERROR_NOT_FOUND as i32) {
                return Err(format!("无法清除 Windows 会话凭据：{error}"));
            }
        }
        Ok(())
    }
}

#[cfg(all(test, windows))]
mod tests {
    use super::*;
    use std::process::Command;
    use std::time::{SystemTime, UNIX_EPOCH};

    const OLD_TOKEN: &str = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA";
    const NEW_TOKEN: &str = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB";

    #[test]
    fn windows_credentials_survive_process_restart() {
        const TEST_SCOPE: &str = "ZWOA_CREDENTIAL_TEST_SCOPE";
        if let Ok(scope) = std::env::var(TEST_SCOPE) {
            let session = DesktopSession::default();
            assert_eq!(session.read(&scope).unwrap().as_deref(), Some(NEW_TOKEN));
            assert!(!session.clear_if_current(&scope, OLD_TOKEN).unwrap());
            assert!(session.clear_if_current(&scope, NEW_TOKEN).unwrap());
            return;
        }

        let nonce = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        let scope = format!("test://{}/{nonce}", std::process::id());
        let session = DesktopSession::default();
        session.store(&scope, OLD_TOKEN).unwrap();
        session.store(&scope, NEW_TOKEN).unwrap();
        let other_scope = session.read(&scope.to_uppercase());

        let child = Command::new(std::env::current_exe().unwrap())
            .args([
                "--exact",
                "credentials::tests::windows_credentials_survive_process_restart",
            ])
            .env(TEST_SCOPE, &scope)
            .output();
        let current = session.read(&scope);
        session.clear_if_current(&scope, NEW_TOKEN).unwrap();

        let child = child.unwrap();
        assert!(
            child.status.success(),
            "{}",
            String::from_utf8_lossy(&child.stdout)
        );
        assert_eq!(current.unwrap(), None);
        assert_eq!(other_scope.unwrap(), None);
    }
}

#[cfg(windows)]
fn wide_target(target: &str) -> Vec<u16> {
    target.encode_utf16().chain(std::iter::once(0)).collect()
}

#[cfg(not(windows))]
#[derive(Default)]
struct Credentials(HashMap<String, String>);

#[cfg(not(windows))]
impl Credentials {
    fn read(&self, target: &str) -> Result<Option<String>, String> {
        Ok(self.0.get(target).cloned())
    }

    fn store(&mut self, target: &str, token: &str) -> Result<(), String> {
        self.0.insert(target.to_owned(), token.to_owned());
        Ok(())
    }

    fn remove(&mut self, target: &str) -> Result<(), String> {
        self.0.remove(target);
        Ok(())
    }
}
