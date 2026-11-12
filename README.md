# 中网联合 OA / CRM

zwOA 是面向公司内部使用的办公与客户管理平台。本项目为公开可见的闭源项目。

当前可运行工作台、模块导航和后端连接状态。客户管理、办公审批、组织与权限已经划分页面入口，业务接口、登录和数据持久化待接入。

## 项目结构

| 路径 | 职责 |
| --- | --- |
| `cmd/api` | GoFrame API 进程入口 |
| `internal/platform/httpserver` | HTTP 配置、健康检查和跨域策略 |
| `internal/modules/workbench` | 工作台业务接口 |
| `apps/web` | React、TypeScript、Ant Design 共享界面，供浏览器和桌面端使用 |
| `apps/desktop` | Tauri 2 桌面窗口和 Windows 打包配置 |

后端按业务划分 Go 包，在一个服务进程内运行。后续可在 `internal/modules` 增加 `crm`、`approvals`、`organization`，并通过 `/api/v1` 暴露接口。数据访问与授权应留在后端，客户端共享 HTTP 契约。将来接入 Android 和 iOS 时，可复用 API，并按设备验证交互和平台能力。

## 本地开发

需要 Go 1.26、Node.js 22.12 或更高版本及 npm。Go 版本由 `go.mod` 声明。

1. 在仓库根目录启动 API。

   ```powershell
   go run ./cmd/api
   ```

2. 在另一个终端启动网页。

   ```powershell
   npm --prefix apps/web ci
   npm --prefix apps/web run dev
   ```

3. 打开 `http://127.0.0.1:5273`。页面显示「服务在线」时，表示工作台接口已连通。

API 默认监听 `127.0.0.1:8720`。`GET /healthz` 用于确认进程能处理请求，`GET /api/v1/workbench/overview` 返回系统名称和版本。设置 `ZWOA_ENABLE_OPENAPI=1` 后，可访问 `GET /openapi.json` 查看接口定义。

使用 `ZWOA_API_ADDR` 可调整 API 监听地址。若开发端口改变，用 `ZWOA_DEV_API_TARGET` 指定 Vite 代理目标。Web 生产构建执行 `npm --prefix apps/web run build`，产物在 `apps/web/dist`。部署时可将同源 `/api` 转发到 Go 服务，或构建时用 `VITE_API_BASE_URL` 指定 API 地址。

## Windows 桌面端

需要 Rust 的 Windows MSVC 工具链、Visual C++ Build Tools、Windows SDK 和 WebView2 Runtime。桌面端运行同一套界面，并访问独立启动的 Go API。

```powershell
npm --prefix apps/web ci
npm --prefix apps/desktop ci
npm --prefix apps/desktop run dev
```

桌面开发命令会启动 `5273` 端口的 Vite 服务。可以用浏览器打开同一个地址。已有网页开发服务占用此端口时，先结束该服务再启动桌面开发命令。

```powershell
npm --prefix apps/desktop run build
```

发布构建生成 Windows 可执行文件和 NSIS 安装包，位于 `apps/desktop/src-tauri/target/release` 和 `apps/desktop/src-tauri/target/release/bundle/nsis`。本地打包默认连接 `http://127.0.0.1:8720`。部署到公司环境时，用 `VITE_API_BASE_URL` 设置实际的 HTTPS API 地址，并把该来源加入 `apps/desktop/src-tauri/tauri.conf.json` 中的 `connect-src`。

API 使用 `ZWOA_ALLOWED_ORIGINS` 控制跨域来源，以逗号分隔完整的协议、主机和端口。Windows 安装包的 WebView 默认来源为 `http://tauri.localhost`，已包含在本地默认配置中。

## 提交约定

使用 Gitmoji 三段式标题 `表情 (模块): 简述`，标题后空一行，用简短的分点正文说明具体改动。按模块或独立修改点提交。详见 [开发协作](CONTRIBUTING.md)。
