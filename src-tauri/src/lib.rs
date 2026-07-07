use tauri::Manager;
use std::process::{Child, Command};
use std::sync::Mutex;
use std::time::Duration;

struct GoServer(Mutex<Option<Child>>);

fn is_port_open(host: &str, port: u16) -> bool {
    use std::net::TcpStream;
    let addr = format!("{}:{}", host, port);
    TcpStream::connect_timeout(
        &addr.parse().expect("invalid address"),
        Duration::from_millis(500),
    )
    .is_ok()
}

fn locate_go_binary(app: &tauri::AppHandle) -> std::path::PathBuf {
    // Try resource_dir (where Tauri puts bundled resources)
    if let Ok(resource_dir) = app.path().resource_dir() {
        #[cfg(target_os = "macos")]
        {
            // On macOS, .app/Contents/Resources/ is resource_dir
            // We use bin/gitboard layout: place Go binary next to tauri binary
            let macos_dir = resource_dir.parent().and_then(|p| p.parent()).map(|p| p.join("MacOS"));
            if let Some(macos_dir) = macos_dir {
                let candidate = macos_dir.join("gitboard");
                if candidate.exists() {
                    return candidate;
                }
            }
        }
        #[cfg(not(target_os = "macos"))]
        {
            let candidate = resource_dir.join("gitboard");
            if candidate.exists() {
                return candidate;
            }
        }
    }
    // Fallback: alongside the running tauri binary
    if let Ok(exe) = std::env::current_exe() {
        if let Some(parent) = exe.parent() {
            let candidate = parent.join("gitboard");
            if candidate.exists() {
                return candidate;
            }
        }
    }
    std::path::PathBuf::from("gitboard")
}

#[tauri::command]
fn server_ready() -> bool {
    is_port_open("127.0.0.1", 28731)
}

#[tauri::command]
fn kill_server(state: tauri::State<GoServer>) -> bool {
    if let Some(mut child) = state.0.lock().unwrap().take() {
        let _ = child.kill();
        let _ = child.wait();
        true
    } else {
        false
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(GoServer(Mutex::new(None)))
        .setup(|app| {
            let bin_path = locate_go_binary(&app.handle());
            eprintln!("[gitboard-tauri] launching backend: {:?}", bin_path);

            let child = Command::new(&bin_path)
                .args(["--port", "28731"])
                .stdout(std::process::Stdio::piped())
                .stderr(std::process::Stdio::piped())
                .spawn()
                .unwrap_or_else(|e| {
                    panic!("failed to launch gitboard backend at {:?}: {}", bin_path, e);
                });

            let state: tauri::State<GoServer> = app.state();
            *state.0.lock().unwrap() = Some(child);

            // Wait up to 10s for the server to start listening
            for i in 0..100 {
                if is_port_open("127.0.0.1", 28731) {
                    eprintln!("[gitboard-tauri] backend ready after {}ms", i * 100);
                    break;
                }
                std::thread::sleep(Duration::from_millis(100));
            }

            // Navigate the webview to the backend
            if let Some(window) = app.get_webview_window("main") {
                let _ = window.eval("window.location.replace('http://127.0.0.1:28731')");
            }

            Ok(())
        })
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::Destroyed = event {
                if let Some(state) = window.app_handle().try_state::<GoServer>() {
                    if let Some(mut child) = state.0.lock().unwrap().take() {
                        let _ = child.kill();
                        let _ = child.wait();
                    }
                }
            }
        })
        .invoke_handler(tauri::generate_handler![server_ready, kill_server])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
